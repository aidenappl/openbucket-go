package routers

import (
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aidenappl/openbucket-go/metadata"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/gorilla/mux"
)

// ObjectList defines the structure of the XML response for listing objects in a bucket.
type ObjectList struct {
	XMLName  xml.Name        `xml:"ListBucketResult"`
	Contents []ObjectContent `xml:"Contents"`
}

type ObjectContent struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	CreatedAt    time.Time `xml:"CreatedAt"`
	ETag         string    `xml:"ETag"`
	Size         int64     `xml:"Size"`
}

func HandleListObjects(w http.ResponseWriter, r *http.Request) {
	// Get the bucket name from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Define the path to the bucket's objects (files)
	bucketDir := filepath.Join("buckets", bucket)

	// Check if the bucket directory exists
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		responder.SendXML(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", "", "")
		log.Println("Bucket not found:", bucketDir)
		return
	} else if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to access bucket", "", "")
		log.Println("Error accessing bucket:", err)
		return
	}

	files, err := os.ReadDir(bucketDir)
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to read bucket directory", "", "")
		log.Println("Error reading bucket directory:", err)
		return
	}

	// Create an ObjectList to hold the response data
	var objectList ObjectList

	// Iterate over the files and add them to the object list
	for _, file := range files {
		// Skip directories and metadata files (.obmeta)
		if file.IsDir() || strings.HasSuffix(file.Name(), ".obmeta") {
			continue
		}

		// Check if the .obmeta file exists
		metadataFilePath := filepath.Join(bucketDir, file.Name()+".obmeta")
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
			var metadata metadata.Metadata
			decoder := xml.NewDecoder(metadataFile)
			if err := decoder.Decode(&metadata); err != nil {
				responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Error parsing metadata XML", "", "")
				log.Println("Error parsing .obmeta XML:", err)
				return
			}

			// Get file info
			info, err := file.Info()
			if err != nil {
				responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to get file info", "", "")
				log.Println("Error getting file info:", err)
				return
			}

			// Append file info to object list
			objectList.Contents = append(objectList.Contents, ObjectContent{
				Key:          file.Name(),
				LastModified: metadata.LastModified,
				CreatedAt:    metadata.UploadedAt,
				ETag:         metadata.ETag,
				Size:         info.Size(),
			})
		} else {
			// If .obmeta file is missing, continue to next file
			continue
		}
	}

	// Set the response header to XML
	w.Header().Set("Content-Type", "application/xml")
	// Write the XML response
	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(objectList)
}
