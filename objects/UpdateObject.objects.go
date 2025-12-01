package objects

import (
	"log"
	"time"

	"github.com/aidenappl/openbucket-go/db"
)

type UpdateObjectRequest struct {
	Public *bool
}

func UpdateObject(bucketID int, objectKey string, req UpdateObjectRequest) error {
	updateMap := make(map[string]interface{})

	if req.Public != nil {
		updateMap["public"] = *req.Public
	}

	updateMap["last_modified"] = time.Now().UTC()

	_, err := db.Psql.Update("objects").SetMap(updateMap).From("buckets").Where("objects.bucket_id = buckets.id").
		Where("buckets.id = ?", bucketID).
		Where("objects.key = ?", objectKey).
		Exec()
	if err != nil {
		log.Println("Error updating object:", err)
		return err
	}
	return nil
}
