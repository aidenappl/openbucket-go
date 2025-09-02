package bucket

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
)

func CreateBucket(bucketName string, owner int) error {
	// Does it already exist?
	b, err := GetBucket(bucketName)
	if err != nil {
		log.Println("Error getting bucket:", err)
		return fmt.Errorf("error getting bucket: %w", err)
	}

	if b != nil {
		log.Println("Bucket already exists:", bucketName)
		return fmt.Errorf("bucket already exists: %s", bucketName)
	}

	// Create bucket
	query := db.Psql.Insert("buckets").Columns(
		"name",
		"owner_id",
		"creation_date",
		"acl",
	).Values(
		bucketName,
		owner,
		time.Now(),
		types.BUCKET_ACLPrivate,
	).Suffix("RETURNING id")

	// Execute the query
	var bucketID int
	err = query.QueryRow().Scan(&bucketID)
	if err != nil {
		return fmt.Errorf("create bucket %s: %w", bucketName, err)
	}

	// Grant owner full permissions
	err = NewGrant(NewGrantReq{
		BucketID:   bucketID,
		GranteeID:  owner,
		Permission: types.FULL_CONTROL,
	})
	if err != nil {
		return fmt.Errorf("failed to save new grant: %v", err)
	}

	// Create local file
	if !util.BucketExists(bucketName) {
		// Create the local directory for the bucket
		err = CreateBucketDirectory(bucketName)
		if err != nil {
			return fmt.Errorf("failed to create local bucket directory: %v", err)
		}
	}

	return nil
}

func CreateBucketDirectory(bucketName string) error {
	// Create the local directory for the bucket
	err := os.MkdirAll(filepath.Join("buckets", bucketName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create local bucket directory: %v", err)
	}
	return nil
}
