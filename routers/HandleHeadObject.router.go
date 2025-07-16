package routers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/gorilla/mux"
)

func HandleHeadObject(w http.ResponseWriter, r *http.Request) {
	// Get the bucket and key (file name) from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	// Validate bucket and key
	if bucket == "" || key == "" {
		http.Error(w, "Bucket and key must be provided", http.StatusBadRequest)
		log.Println("Bucket or key is empty")
		return
	}

	// Define the file path for the object in the bucket
	filePath := filepath.Join("buckets", bucket, key)

	// Check if the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		responder.SendAccessDeniedXML(w, nil, nil)
		log.Println("File not found:", filePath)
		return
	} else if err != nil {
		// If there's any other error, return 500 Internal Server Error
		responder.SendAccessDeniedXML(w, nil, nil)
		log.Println("Error accessing file:", err)
		return
	}

	// Retrieve file info for metadata
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		responder.SendAccessDeniedXML(w, nil, nil)
		log.Println("Error getting file info:", err)
		return
	}

	metadata := middleware.RetrieveMetadata(r)
	if metadata == nil {
		responder.SendAccessDeniedXML(w, nil, nil)
		log.Println("Metadata not found for file:", filePath)
		return
	}

	// Set the appropriate headers for the HEAD response
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat)) // Last-Modified header
	w.Header().Set("Content-Type", "application/octet-stream")                        // Content-Type header (can be more specific)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))              // Content-Length header
	w.Header().Set("ETag", metadata.ETag)

	// Respond with status OK (200)
	w.WriteHeader(http.StatusOK)
	log.Println("Metadata retrieved for file:", filePath)
}
