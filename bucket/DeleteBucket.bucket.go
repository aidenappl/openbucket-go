package bucket

import (
	"fmt"
	"os"

	"github.com/aidenappl/openbucket-go/util"
)

// Delete Bucket is a destructive handler to removeAll files at a specified bucket
func DeleteBucket(bucketName string) error {
	// validate bucketName
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	// lookup bucket
	if !util.BucketExists(bucketName) {
		return fmt.Errorf("bucket %s does not exist", bucketName)
	}

	// Delete metadata file
	err := os.Remove(util.GetBucketMetadataPath(bucketName))
	if err != nil {
		return fmt.Errorf("error deleting metadata file for bucket %s: %w", bucketName, err)
	}

	// delete bucket
	return os.RemoveAll(util.GetBucketPath(bucketName))
}
