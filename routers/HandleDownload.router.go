package routers

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

// Metadata represents the structure of the metadata XML file.
type Metadata struct {
	ETag         string `xml:"etag"`
	Bucket       string `xml:"bucket"`
	Key          string `xml:"key"`
	Tags         string `xml:"tags"`
	VersionID    string `xml:"versionId"`
	Owner        string `xml:"owner"`
	Public       bool   `xml:"public"`
	LastModified string `xml:"lastModified"`
	UploadedAt   string `xml:"uploadedAt"`
}

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	// Get the bucket and key (file name) from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	// Validate bucket and key
	if bucket == "" || key == "" {
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Bucket and key must be provided", "", "")
		log.Println("Bucket or key is empty")
		return
	}

	// Validate key is not obmeta
	if strings.HasSuffix(key, ".obmeta") {
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Cannot access metadata files directly", "", "")
		log.Println("Attempted to access metadata file directly:", key)
		return
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
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Retrieve the file info to get LastModified
	fileInfo, err := file.Stat()
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to get file info", "", "")
		log.Println("Error getting file info:", err)
		return
	}

	// Check if the .obmeta file exists
	metadataFilePath := filepath.Join("buckets", bucket, key+".obmeta")
	var metadata Metadata
	if _, err := os.Stat(metadataFilePath); err == nil {
		// If .obmeta exists, read and parse the metadata XML
		metadataFile, err := os.Open(metadataFilePath)
		if err != nil {
			responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to open .obmeta file", "", "")
			log.Println("Error opening .obmeta file:", err)
			return
		}
		defer metadataFile.Close()

		// Parse the XML metadata
		decoder := xml.NewDecoder(metadataFile)
		if err := decoder.Decode(&metadata); err != nil {
			responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Error parsing metadata XML", "", "")
			log.Println("Error parsing .obmeta XML:", err)
			return
		}

		// Validate public access
		if !metadata.Public {
			responder.SendXML(w, http.StatusUnauthorized, "Unauthorized", "You must be authorized to access this file", "", "")
			log.Println("Unauthorized access attempt for file:", filePath)
			return
		}

		// Set metadata in response headers (optional)
		w.Header().Set("X-Object-ETag", metadata.ETag)
		w.Header().Set("X-Object-LastModified", metadata.LastModified)
		w.Header().Set("X-Object-Owner", metadata.Owner)
		w.Header().Set("X-Object-Public", fmt.Sprintf("%t", metadata.Public))
	} else {
		responder.SendXML(w, http.StatusNotFound, "MetadataNotFound", "Metadata file not found", "", "")
		log.Println("Metadata file not found:", metadataFilePath)
		return
	}

	// Validate the presigned URL signature (Optional)
	if !isValidPresignURL(r, bucket, key) {
		responder.SendXML(w, http.StatusUnauthorized, "InvalidSignature", "The presigned URL is invalid or expired", "", "")
		log.Println("Invalid or expired presigned URL:", key)
		return
	}

	// Set the Content-Type header to serve the file
	w.Header().Set("Content-Type", tools.ContentType(filePath))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat)) // Set the LastModified header

	// Copy the file content to the response
	_, err = io.Copy(w, file)
	if err != nil {
		// If there was an error copying the file, return 500 Internal Server Error
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Error transferring file", "", "")
		log.Println("Error transferring file:", err)
		return
	}

	// Successfully served the file
	log.Println("File successfully served:", filePath)
}

func isValidPresignURL(r *http.Request, bucket, key string) bool {
	// Example function to validate the pre-signed URL (simplified for illustration)
	// Extract parameters from URL query
	amzDate := r.URL.Query().Get("X-Amz-Date")
	signature := r.URL.Query().Get("X-Amz-Signature")
	if amzDate == "" || signature == "" {
		log.Println("Missing required parameters for presigned URL validation")
		return false
	}
	// Validate signature, expiration, etc.
	// You can use AWS SDK for Go to verify this signature, expiration, and other parameters

	// For now, return true to proceed with validation logic
	// Implement actual signature verification logic here based on your use case.
	// You could use tools from the AWS SDK to verify the signature and expiration.
	return true
}
