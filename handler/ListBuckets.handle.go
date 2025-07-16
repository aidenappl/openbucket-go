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

	bucketList := &types.BucketList{
		Buckets: struct {
			Bucket []types.Bucket `xml:"Bucket"`
		}{
			Bucket: make([]types.Bucket, len(*buckets)),
		},
	}

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

	var bucketList []types.Bucket

	for _, file := range files {
		if file.IsDir() {

			info, err := file.Info()
			if err != nil {
				log.Println("Error getting file info:", err)
				continue
			}
			bucketList = append(bucketList, types.Bucket{
				Name:         file.Name(),
				CreationDate: info.ModTime().Format(time.RFC3339),
			})
		}
	}
	return &bucketList, nil
}
