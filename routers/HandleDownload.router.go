package routers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	// Get the bucket and key (file name) from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	// Get request metadata
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Validate bucket and key
	if bucket == "" || key == "" {
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Bucket and key must be provided", "", "")
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	// Get request permission state
	permissions := middleware.RetrievePermissions(r)

	// Get metadata for the object
	metadata := middleware.RetrieveMetadata(r)

	// Check if there is a session (if applicable)
	session := middleware.RetrieveSession(r)

	// Validate the presigned URL signature (Optional only on restricted access)
	if !metadata.Public && !permissions.AllowGlobalRead && session == nil {
		if !isValidPresignURL(r, bucket, key) {
			responder.SendXML(w, http.StatusUnauthorized, "InvalidSignature", "The presigned URL is invalid or expired", "", "")
			log.Println(request, host, "Invalid or expired presigned URL:", key)
			return
		}
	}

	// Define the file path for the object in the bucket
	filePath := filepath.Join("buckets", bucket, key)

	// Check if the file exists
	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		responder.SendXML(w, http.StatusNotFound, "AccessDenied", "Access Denied", "", "")
		return
	} else if err != nil {
		// If there's any other error, return 500 Internal Server Error
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to retrieve file", "", "")
		log.Println(request, host, "Error opening file:", err)
		return
	}
	defer file.Close()

	// Retrieve the file info to get LastModified
	fileInfo, err := file.Stat()
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to get file info", "", "")
		log.Println(request, host, "Error getting file info:", err)
		return
	}

	// Set metadata in response headers (optional)
	w.Header().Set("ETag", metadata.ETag)
	w.Header().Set("Content-Type", tools.ContentType(filePath))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat)) // Set the LastModified header

	// Copy the file content to the response
	_, err = io.Copy(w, file)
	if err != nil {
		// If there was an error copying the file, return 500 Internal Server Error
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Error transferring file", "", "")
		log.Println(request, host, "Error transferring file:", err)
		return
	}

	// Successfully served the file
	log.Println("File successfully served:", filePath)
}

// TODO: Implement with AWS Signature
func isValidPresignURL(r *http.Request, bucket, key string) bool {
	// Example function to validate the pre-signed URL (simplified for illustration)
	// Extract parameters from URL query
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)
	amzDate := r.URL.Query().Get("X-Amz-Date")
	signature := r.URL.Query().Get("X-Amz-Signature")
	if amzDate == "" || signature == "" {
		log.Println(request, host, "Missing required parameters for presigned URL validation")
		return false
	}
	return true
}
