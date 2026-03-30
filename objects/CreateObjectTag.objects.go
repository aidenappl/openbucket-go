package objects

import (
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

type CreateObjectTagReq struct {
	BucketID int    `json:"bucket_id"`
	ObjectID int    `json:"object_id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

func CreateObjectTag(req CreateObjectTagReq) error {
	_, err := db.Psql.Insert("object_tags").
		Columns("bucket_id", "object_id", "tag_key", "tag_value").
		Values(req.BucketID, req.ObjectID, req.Key, req.Value).
		Exec()
	return err
}

func BulkCreateObjectTags(bucketID int, objectID int, tags []types.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	// Use multi-row INSERT for efficiency (single round-trip instead of N)
	insert := db.Psql.Insert("object_tags").
		Columns("bucket_id", "object_id", "tag_key", "tag_value")

	for _, tag := range tags {
		insert = insert.Values(bucketID, objectID, tag.Key, tag.Value)
	}

	_, err := insert.Exec()
	return err
}
