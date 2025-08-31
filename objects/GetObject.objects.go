package objects

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

func GetObject(bucketName string, key string) (*types.ObjectMetadata, error) {
	rows, err := db.Psql.Select(
		"objects.id",
		"objects.bucket_id",
		"objects.key",
		"objects.etag",
		"objects.version_id",
		"objects.owner_id",
		"objects.public",
		"objects.size",
		"objects.last_modified",
		"objects.uploaded_at",
	).
		From("objects").
		Join("buckets ON objects.bucket_id = buckets.id").
		Where(sq.Eq{"objects.key": key, "buckets.name": bucketName}).
		Query()
	if err != nil {
		return nil, fmt.Errorf("error querying object metadata: %w", err)
	}
	defer rows.Close()
	var obj types.ObjectMetadata
	for rows.Next() {
		if err := rows.Scan(
			&obj.ID,
			&obj.BucketID,
			&obj.Key,
			&obj.ETag,
			&obj.VersionID,
			&obj.OwnerID,
			&obj.Public,
			&obj.Size,
			&obj.LastModified,
			&obj.UploadedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning object metadata: %w", err)
		}
	}
	return &obj, nil
}
