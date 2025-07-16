package routers

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/metadata"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

func HandleUpload(w http.ResponseWriter, r *http.Request) {
	// Get the bucket and key (file/folder name) from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	// Validate bucket and key
	if bucket == "" || key == "" {
		http.Error(w, "Bucket and key must be provided", http.StatusBadRequest)
		log.Println("Bucket or key is empty")
		return
	}

	// Get the user session
	user := middleware.RetrieveSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Unauthorized access attempt")
		return
	}

	// Define the destination path for the file/folder in the server's local storage
	filePath := filepath.Join("buckets", bucket, key)

	// Check if the bucket directory exists
	bucketDir := filepath.Join("buckets", bucket)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		http.Error(w, "Bucket not found", http.StatusNotFound)
		log.Println("Bucket not found:", bucketDir)
		return
	} else if err != nil {
		http.Error(w, "Unable to access bucket", http.StatusInternalServerError)
		log.Println("Error accessing bucket:", err)
		return
	}

	// Check if it's a directory upload (if key has no file extension, it is a directory)
	isDirectory := strings.HasSuffix(key, "/")
	if isDirectory {
		// If the key represents a directory, ensure the directory exists
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			log.Println("Error creating directory:", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		log.Println("Directory created:", filePath)
		return
	}

	// If it's a file, create the directories and the file
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		log.Println("Error creating directory:", err)
		return
	}

	// Create the file on the server
	file, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		log.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Copy the file contents from the request body to the server file
	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		log.Println("Error saving file:", err)
		return
	}

	// Generate the ETag after file upload
	etag, err := tools.GenerateETag(filePath)
	if err != nil {
		http.Error(w, "Error generating ETag", http.StatusInternalServerError)
		log.Println("Error generating ETag:", err)
		return
	}

	metadata := metadata.New(bucket, key, etag, false, user.KeyID)

	metadataFilePath := filePath + ".obmeta"
	metadataFile, err := os.Create(metadataFilePath)
	if err != nil {
		http.Error(w, "Error saving metadata", http.StatusInternalServerError)
		log.Println("Error saving metadata:", err)
		return
	}
	defer metadataFile.Close()

	// Write the metadata to the file
	metadataXML, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		log.Println("Error marshalling metadata to XML:", err)
		http.Error(w, "Error marshalling metadata", http.StatusInternalServerError)
		return
	}

	_, err = metadataFile.WriteString(string(metadataXML))
	if err != nil {
		log.Println("Error writing to metadata file:", err)
		http.Error(w, "Error writing to metadata file", http.StatusInternalServerError)
		return
	}

	// Respond with success (status 200 OK)
	w.WriteHeader(http.StatusOK)
	log.Println("File uploaded successfully. ETag:", etag)
}
