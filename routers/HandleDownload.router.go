package routers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {

	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	q := r.URL.Query()
	if _, ok := q["acl"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "ACL query parameter is not supported for download")
		return
	}
	if _, ok := q["tagging"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Tagging query parameter is not supported for download")
		return
	}
	if _, ok := q["uploadId"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "UploadId query parameter is not supported for download")
		return
	}
	if _, ok := q["attributes"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "UploadId query parameter is not supported for download")
		return
	}
	if _, ok := q["legal-hold"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "LegalHold query parameter is not supported for download")
		return
	}
	if _, ok := q["retention"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Retention query parameter is not supported for download")
		return
	}
	if _, ok := q["torrent"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Torrent query parameter is not supported for download")
		return
	}

	handleDownload(w, r)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {

	// Get request variables
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]
	// Get host & request from context
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Validate bucket & key
	if bucket == "" || key == "" {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	// Structure request
	filePath := filepath.Join("buckets", bucket, key)

	// Open requested file
	file, err := os.Open(filePath)
	if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error getting file info:", err)
		return
	}

	// Get file information
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", request, host)
		return
	} else if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error opening file:", err)
		return
	} else if fileInfo.IsDir() {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "File is a directory, not a valid object:", filePath)
		return
	}
	defer file.Close()

	// Check user permissions for file
	permissions := middleware.RetrieveBucket(r) // Bucket permissions
	metadata := middleware.RetrieveMetadata(r)  // File metadata
	session := middleware.RetrieveSession(r)    // User session

	if !metadata.Public && !types.IsBucketACLRead(permissions.ACL) && session == nil {
		if !isValidPresignURL(r, bucket, key) {
			responder.SendAccessDeniedXML(w, &request, &host)
			log.Println(request, host, "Invalid or expired presigned URL:", key)
			return
		}

		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Access denied for bucket:", bucket, "key:", key)
		return
	}

	// Structure response headers
	w.Header().Set("ETag", metadata.ETag.ToString())
	w.Header().Set("X-Amz-Meta-owner-id", metadata.Owner.ID)
	w.Header().Set("X-Amz-Meta-owner-display-name", metadata.Owner.DisplayName)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	w.Header().Set("Content-Type", tools.ContentType(filePath))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("x-amz-tagging-count", strconv.Itoa(len(metadata.Tags.Tag)))
	w.Header().Set("x-amz-version-id", strconv.Itoa(metadata.VersionID))

	// Transfer file content
	_, err = io.Copy(w, file)
	if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error transferring file:", err)
		return
	}

	log.Println("File successfully served:", filePath)
}

func isValidPresignURL(r *http.Request, bucket, key string) bool {
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
