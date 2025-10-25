package objects

import (
	"fmt"
	"log"
	"time"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
)

func CreateFolder(filePath string, key string, bucket types.Bucket, user types.Authorization) error {
	etag, err := tools.GenerateETag(filePath)
	if err != nil {
		log.Println("Error generating ETag:", err)
		return err
	}

	_, err = db.Psql.Insert("objects").Columns(
		"bucket_id",
		"etag",
		"key",
		"owner_id",
		"public",
		"last_modified",
		"uploaded_at",
		"version_id",
		"size",
	).Values(
		bucket.ID,
		etag,
		key,
		user.ID,
		false,
		time.Now(),
		time.Now(),
		1,
		0,
	).Exec()
	if err != nil {
		log.Println("Error inserting object metadata:", err)
		return fmt.Errorf("failed to create folder: %w", err)
	}

	return nil
}
