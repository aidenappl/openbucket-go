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

	// Use multi-row INSERT for efficiency (single round-trip instead of N)
	insert := db.Psql.Insert("bucket_tags").
		Columns("bucket_id", "tag_key", "tag_value")

	for _, tag := range tags {
		insert = insert.Values(bucketID, tag.Key, tag.Value)
	}

	_, err := insert.Exec()
	return err
}
