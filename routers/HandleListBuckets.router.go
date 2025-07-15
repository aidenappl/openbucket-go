package routers

import (
	"encoding/xml"
	"log"
	"net/http"
	"os"
)

type BucketList struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Buckets struct {
		Bucket []struct {
			Name         string `xml:"Name"`
			CreationDate string `xml:"CreationDate"`
		} `xml:"Bucket"`
	} `xml:"Buckets"`
}

func HandleListBuckets(w http.ResponseWriter, r *http.Request) {
	// Read the buckets directory to list all bucket directories
	bucketsDir := "buckets"
	files, err := os.ReadDir(bucketsDir)
	if err != nil {
		http.Error(w, "Unable to read buckets directory", http.StatusInternalServerError)
		log.Println("Error reading buckets directory:", err)
		return
	}

	// Create a BucketList object to hold the response
	var bucketList BucketList

	// Add each directory name as a bucket
	for _, file := range files {
		if file.IsDir() {
			// Add a new bucket to the list
			bucketList.Buckets.Bucket = append(bucketList.Buckets.Bucket, struct {
				Name         string `xml:"Name"`
				CreationDate string `xml:"CreationDate"`
			}{
				Name:         file.Name(),
				CreationDate: "2025-07-15T00:00:00.000Z", // Mocked creation date
			})
		}
	}

	// Set the Content-Type to XML
	w.Header().Set("Content-Type", "application/xml")
	// Write the response as XML
	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(bucketList)
}
