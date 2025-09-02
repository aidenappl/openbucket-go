package bucket

import (
	"fmt"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
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

	// Delete from DB
	// Delete Bucket Permissions
	bucket, err := GetBucket(bucketName)
	if err != nil {
		return fmt.Errorf("get bucket: %w", err)
	}
	_, err = db.Psql.Delete("bucket_permissions").
		Where(sq.Eq{"bucket_id": bucket.ID}).Exec()
	if err != nil {
		return fmt.Errorf("delete bucket permissions from db: %w", err)
	}

	// Delete Bucket
	_, err = db.Psql.Delete("buckets").Where(sq.Eq{"name": bucketName}).Exec()
	if err != nil {
		return fmt.Errorf("delete bucket from db: %w", err)
	}

	// delete bucket
	return os.RemoveAll(util.GetBucketPath(bucketName))
}
