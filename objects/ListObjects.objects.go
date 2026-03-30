package objects

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// ListObjectsOptions configures the ListObjects query
type ListObjectsOptions struct {
	BucketID int // If provided, skips bucket lookup
}

func ListObjects(bucketName string, opts ...*ListObjectsOptions) ([]types.ObjectMetadata, error) {
	var bucketID int

	// Check if bucket ID was provided to skip lookup
	if len(opts) > 0 && opts[0] != nil && opts[0].BucketID > 0 {
		bucketID = opts[0].BucketID
	} else {
		// Get the bucket & validate it exists
		b, err := bucket.GetBucket(bucketName, &bucket.GetBucketOptions{IncludeGrants: false})
		if err != nil {
			return nil, fmt.Errorf("failed to get bucket %q: %w", bucketName, err)
		}

		// If bucket is nil, then it does not exist
		if b == nil {
			return nil, fmt.Errorf("bucket %q not found", bucketName)
		}
		bucketID = b.ID
	}

	var out []types.ObjectMetadata

	// List all objects within a bucket using bucket ID for efficiency
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

		"authorizations.key_id",
		"authorizations.name",
	).
		From("objects").
		Join("authorizations ON objects.owner_id = authorizations.id").
		Where(sq.Eq{"objects.bucket_id": bucketID})

	// Any future conditionals can go below:

	// Run the query
	rows, err := query.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %q: %w", bucketName, err)
	}

	defer rows.Close()

	// Scan through the rows
	for rows.Next() {
		var obj types.ObjectMetadata
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

			&obj.Owner.ID,
			&obj.Owner.DisplayName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan object row: %w", err)
		}
		out = append(out, obj)
	}

	return out, nil
}
