package bucket

import (
	"fmt"
	"os"

	"github.com/aidenappl/openbucket-go/util"
)

// Delete Bucket is a destructive handler to removeAll files at a specified bucket
func DeleteBucket(bucketName string) error {
	// TODO: Repair
	// validate bucketName
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	// lookup bucket
	if !util.BucketExists(bucketName) {
		return fmt.Errorf("bucket %s does not exist", bucketName)
	}

	// delete bucket
	return os.RemoveAll(util.GetBucketPath(bucketName))
}
