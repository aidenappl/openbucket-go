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
	for _, tag := range tags {
		err := CreateObjectTag(CreateObjectTagReq{
			BucketID: bucketID,
			ObjectID: objectID,
			Key:      tag.Key,
			Value:    tag.Value,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
