package handler

import (
	"log"
	"os"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

func ListBucketsXML() (*types.BucketList, error) {
	buckets, err := ListBuckets()
	if err != nil {
		log.Println("Error listing buckets:", err)
		return nil, err
	}

	// Create a BucketList object to hold the response
	bucketList := &types.BucketList{
		Buckets: struct {
			Bucket []types.Bucket `xml:"Bucket"`
		}{
			Bucket: make([]types.Bucket, len(*buckets)),
		},
	}

	// Populate the BucketList with the bucket information
	copy(bucketList.Buckets.Bucket, *buckets)

	return bucketList, nil
}

func ListBuckets() (*[]types.Bucket, error) {
	bucketsDir := "buckets"
	files, err := os.ReadDir(bucketsDir)
	if err != nil {
		log.Println("Error reading buckets directory:", err)
		return nil, err
	}

	// Create a BucketList object to hold the response
	var bucketList []types.Bucket

	// Add each directory name as a bucket
	for _, file := range files {
		if file.IsDir() {
			// Add a new bucket to the list
			info, err := file.Info()
			if err != nil {
				log.Println("Error getting file info:", err)
				continue
			}
			bucketList = append(bucketList, types.Bucket{
				Name:         file.Name(),
				CreationDate: info.ModTime().Format(time.RFC3339), // Use actual creation date
			})
		}
	}
	return &bucketList, nil
}
