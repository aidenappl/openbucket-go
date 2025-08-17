package handler

import (
	"log"
	"os"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/types"
)

func ListBucketsXML(params *ListBucketsParams) (*types.BucketList, error) {
	buckets, err := ListBuckets(params)
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

type ListBucketsParams struct {
	Prefix *string
	Filter *ListBucketsParamFilter
}

type ListBucketsParamFilter struct {
	OwnerID *string
}

func ListBuckets(params *ListBucketsParams) (*[]types.Bucket, error) {
	bucketsDir := "buckets"
	files, err := os.ReadDir(bucketsDir)
	if err != nil {
		log.Println("Error reading buckets directory:", err)
		return nil, err
	}

	var bucketList []types.Bucket

	for _, file := range files {
		if file.IsDir() {

			metadata, err := bucket.RetrieveMetadata(file.Name())
			if err != nil {
				log.Println("Error retrieving metadata:", err)
				continue
			}

			if params.Filter != nil && params.Filter.OwnerID != nil {
				if metadata.Owner.ID != *params.Filter.OwnerID {
					continue
				}
			}

			bucketList = append(bucketList, *metadata)
		}
	}

	return &bucketList, nil
}
