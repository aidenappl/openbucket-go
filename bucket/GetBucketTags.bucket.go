package bucket

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

func GetBucketTags(bucketID int) (*[]types.BucketTag, error) {
	var tags []types.BucketTag
	rows, err := db.Psql.Select(
		"id",
		"tag_key",
		"tag_value",
	).From("bucket_tags").
		Where(sq.Eq{"bucket_id": bucketID}).
		Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var tag types.BucketTag
		if err := rows.Scan(&tag.ID, &tag.Key, &tag.Value); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return &tags, nil
}
