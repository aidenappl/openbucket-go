package objects

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// GetObjectOptions configures what data to load with GetObject
type GetObjectOptions struct {
	IncludeTags bool
}

// DefaultGetObjectOptions returns the default options (backward compatible - includes tags)
func DefaultGetObjectOptions() *GetObjectOptions {
	return &GetObjectOptions{
		IncludeTags: true,
	}
}

func GetObject(bucketName string, key string, etag *string, opts ...*GetObjectOptions) (*types.ObjectMetadata, error) {
	// Use default options if none provided (backward compatible)
	opt := DefaultGetObjectOptions()
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

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

		// Get object tags only if requested
		if opt.IncludeTags {
			tags, err := GetObjectTags(obj.BucketID, obj.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get object tags: %w", err)
			}
			obj.Tags.Tag = tags
		}
		obj.Owner = owner
	}

	if obj.ID == 0 {
		return nil, nil
	}

	return &obj, nil
}
