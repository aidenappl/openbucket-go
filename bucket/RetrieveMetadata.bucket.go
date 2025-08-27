package bucket

import (
	"encoding/xml"
	"log"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
)

func SaveBucketPermissions(bucket string, metadata *types.Bucket) error {
	// atomic write
	tmp := util.GetBucketMetadataPath(bucket) + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return err
	}

	metadata.Xmlns = "http://s3.amazonaws.com/doc/2006-03-01/"

	buf, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	buf = append([]byte(xml.Header), buf...)
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, util.GetBucketMetadataPath(bucket))
}

func RetrieveMetadata(bucketName string) (*types.Bucket, error) {
	// Retrieve metadata for the specified bucket
	metadataFilePath := util.GetBucketMetadataPath(bucketName)
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
