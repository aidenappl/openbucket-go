package objects

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

func GetObject(bucketName string, key string, etag *string) (*types.ObjectMetadata, error) {
	query := db.Psql.Select(
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

		"authorizations.id",
		"authorizations.name",
	).
		From("objects").
		Join("buckets ON objects.bucket_id = buckets.id").
		Join("authorizations ON objects.owner_id = authorizations.id").
		Where(sq.Eq{"objects.key": key, "buckets.name": bucketName})

	if etag != nil {
		query = query.Where(sq.Eq{"objects.etag": *etag})
	}

	rows, err := query.Query()
	if err != nil {
		return nil, fmt.Errorf("error querying object metadata: %w", err)
	}
	defer rows.Close()
	var obj types.ObjectMetadata
	var owner types.UserObject
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

			&owner.ID,
			&owner.DisplayName,
		); err != nil {
			return nil, fmt.Errorf("error scanning object metadata: %w", err)
		}

		// Get object tags
		tags, err := GetObjectTags(obj.BucketID, obj.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get object tags: %w", err)
		}
		obj.Owner = owner
		obj.Tags.Tag = tags
	}

	if obj.ID == 0 {
		return nil, nil
	}

	return &obj, nil
}
