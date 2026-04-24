package objects

import (
	"fmt"
	"os"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
)

// DeleteObject removes an object from a bucket
func DeleteObject(bucket types.Bucket, objectName string) error {
	// Validate path to prevent traversal
	resolvedPath, err := util.ValidateBucketPath(bucket.Name, objectName)
	if err != nil {
		return err
	}

	// check if object or folder
	if strings.HasSuffix(objectName, "/") {
		// Attempt to remove the folder
		err := os.RemoveAll(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to delete object folder: %v", err) // Return the error if deletion fails
		}
	} else {
		// Attempt to remove the object file
		err := os.Remove(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to delete object file: %v", err) // Return the error if deletion fails
		}
	}

	// Get the object
	object, err := GetObject(bucket.Name, objectName, nil)
	if err != nil {
		return fmt.Errorf("failed to get object: %v", err)
	}

	// Remove the object tags from the database
	err = DeleteAllObjectTags(bucket.ID, object.ID)
	if err != nil {
		return fmt.Errorf("failed to delete object tags: %v", err)
	}

	// Remove object from the database
	_, err = db.Psql.
		Delete("objects").
		Where(sq.Eq{"objects.bucket_id": bucket.ID, "objects.key": objectName}).
		Exec()
	if err != nil {
		return fmt.Errorf("failed to delete object from database: %v", err)
	}

	return nil // Return nil if deletion is successful
}

// DeleteObject removes an object from a bucket
func DeleteVersionedObject(bucket types.Bucket, objectName string, versionId string) error {
	return DeleteObject(bucket, objectName) // For simplicity, we treat versioned objects the same as regular objects TODO: implement versioning
}
