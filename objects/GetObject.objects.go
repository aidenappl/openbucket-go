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

	// ⚠️ Read every row and close them BEFORE any other query. Tags used to be
	// fetched inside this loop, holding this connection while asking the pool
	// for a second: every object request (the permissions middleware calls
	// this, CDN downloads included) needed two connections at once, so ~100
	// concurrent requests took all 100 and waited on each other forever. On
	// 2026-10-15 a burst of HEADs deadlocked cdn.trailblaze.to for ~15 hours.
	// TestGetObjectUnderPoolPressure guards this.
	rows, err := query.Query()
	if err != nil {
		return nil, fmt.Errorf("error querying object metadata: %w", err)
	}
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
			rows.Close()
			return nil, fmt.Errorf("error scanning object metadata: %w", err)
		}
		obj.Owner = owner
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading object metadata: %w", err)
	}

	if obj.ID == 0 {
		return nil, nil
	}

	// Tags only now, with the connection above back in the pool.
	if opt.IncludeTags {
		tags, err := GetObjectTags(obj.BucketID, obj.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get object tags: %w", err)
		}
		obj.Tags.Tag = tags
	}

	return &obj, nil
}
