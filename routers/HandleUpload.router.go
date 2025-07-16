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

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	if bucket == "" || key == "" {
		http.Error(w, "Bucket and key must be provided", http.StatusBadRequest)
		log.Println("Bucket or key is empty")
		return
	}

	user := middleware.RetrieveSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Unauthorized access attempt")
		return
	}

	filePath := filepath.Join("buckets", bucket, key)

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

	isDirectory := strings.HasSuffix(key, "/")
	if isDirectory {

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

	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		log.Println("Error creating directory:", err)
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		log.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		log.Println("Error saving file:", err)
		return
	}

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

	w.WriteHeader(http.StatusOK)
	log.Println("File uploaded successfully. ETag:", etag)
}
