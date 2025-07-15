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

	"github.com/aidenappl/openbucket-go/metadata"
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
		fmt.Println("Bucket or key is empty")
		return
	}

	// Validate key is not obmeta
	if strings.HasSuffix(key, ".obmeta") {
		http.Error(w, "Cannot upload metadata files directly", http.StatusBadRequest)
		fmt.Println("Attempted to upload metadata file directly:", key)
		return
	}

	// Define the destination path for the file/folder in the server's local storage
	filePath := filepath.Join("buckets", bucket, key)

	// Check if the bucket directory exists
	bucketDir := filepath.Join("buckets", bucket)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		http.Error(w, "Bucket not found", http.StatusNotFound)
		fmt.Println("Bucket not found:", bucketDir)
		return
	} else if err != nil {
		http.Error(w, "Unable to access bucket", http.StatusInternalServerError)
		fmt.Println("Error accessing bucket:", err)
		return
	}

	// Check if it's a directory upload (if key has no file extension, it is a directory)
	isDirectory := strings.HasSuffix(key, "/")
	if isDirectory {
		// If the key represents a directory, ensure the directory exists
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			fmt.Println("Error creating directory:", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Println("Directory created:", filePath)
		return
	}

	// If it's a file, create the directories and the file
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		fmt.Println("Error creating directory:", err)
		return
	}

	// Create the file on the server
	file, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Copy the file contents from the request body to the server file
	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		fmt.Println("Error saving file:", err)
		return
	}

	// Generate the ETag after file upload
	etag, err := tools.GenerateETag(filePath)
	if err != nil {
		http.Error(w, "Error generating ETag", http.StatusInternalServerError)
		fmt.Println("Error generating ETag:", err)
		return
	}

	metadata := metadata.New(
		bucket,
		key,
		etag,
		false, // Assuming public is false by default, can be changed based on requirements
	)

	metadataFilePath := filePath + ".obmeta"
	metadataFile, err := os.Create(metadataFilePath)
	if err != nil {
		http.Error(w, "Error saving metadata", http.StatusInternalServerError)
		fmt.Println("Error saving metadata:", err)
		return
	}
	defer metadataFile.Close()

	// Write the metadata to the file
	encoder := xml.NewEncoder(metadataFile)
	err = encoder.Encode(metadata)
	if err != nil {
		http.Error(w, "Error encoding metadata", http.StatusInternalServerError)
		fmt.Println("Error encoding metadata:", err)
		return
	}

	// Respond with success (status 200 OK)
	w.WriteHeader(http.StatusOK)
	log.Println("File uploaded successfully. ETag:", etag)
}
