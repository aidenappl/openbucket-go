package bucket

import (
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

func CreateBucketTag(bucketID int, tags types.BucketTag) error {
	_, err := db.Psql.Insert("bucket_tags").
		Columns("bucket_id", "tag_key", "tag_value").
		Values(bucketID, tags.Key, tags.Value).
		Exec()
	return err
}

func BulkCreateBucketTags(bucketID int, tags []types.BucketTag) error {
	if len(tags) == 0 {
		return nil
	}

	// Create a slice of the values to insert
	values := make([]interface{}, 0, len(tags)*3)
	for _, tag := range tags {
		values = append(values, bucketID, tag.Key, tag.Value)
	}

	_, err := db.Psql.Insert("bucket_tags").
		Columns("bucket_id", "tag_key", "tag_value").
		Values(values...).
		Exec()
	return err
}
