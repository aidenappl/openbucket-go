package bucket

import (
	"encoding/xml"
	"log"
	"os"

	"github.com/aidenappl/openbucket-go/types"
)

func RetrieveMetadata(bucketName string) (*types.Bucket, error) {
	// Retrieve metadata for the specified bucket
	metadataFilePath := "buckets/" + bucketName + ".obpermissions"
	file, err := os.Open(metadataFilePath)
	if err != nil {
		log.Println("Error opening metadata file:", err)
		return nil, err
	}
	defer file.Close()

	var metadata types.Bucket
	if err := xml.NewDecoder(file).Decode(&metadata); err != nil {
		log.Println("Error decoding metadata XML:", err)
		return nil, err
	}

	return &metadata, nil
}
