package bucket

import (
	"fmt"
	"log"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// GetBucketOptions configures what data to load with GetBucket
type GetBucketOptions struct {
	IncludeGrants bool
}

// DefaultGetBucketOptions returns the default options (backward compatible - includes grants)
func DefaultGetBucketOptions() *GetBucketOptions {
	return &GetBucketOptions{
		IncludeGrants: true,
	}
}

func GetBucket(bucketName string, opts ...*GetBucketOptions) (*types.Bucket, error) {
	// Use default options if none provided (backward compatible)
	opt := DefaultGetBucketOptions()
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	rows, err := db.Psql.Select(
		"buckets.id",
		"buckets.name",
		"buckets.creation_date",
		"buckets.acl",

		"authorizations.key_id",
		"authorizations.name",
	).From("buckets").
		LeftJoin("authorizations ON buckets.owner_id = authorizations.id").
		Where(sq.Or{
			sq.Eq{"buckets.name": bucketName},
		}).
		Query()
	if err != nil {
		return nil, fmt.Errorf("query buckets: %w", err)
	}
	defer rows.Close()

	var bucket types.Bucket
	if rows.Next() {
		if err := rows.Scan(
			&bucket.ID,
			&bucket.Name,
			&bucket.CreationDate,
			&bucket.ACL,

			&bucket.Owner.ID,
			&bucket.Owner.DisplayName,
		); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}

		// Get the bucket grants only if requested
		if opt.IncludeGrants {
			grants, err := GetBucketGrants(bucketName)
			if err != nil {
				log.Println("Error getting bucket grants:", err)
				return nil, fmt.Errorf("error getting bucket grants: %w", err)
			}
			bucket.Grants = grants
		}

		return &bucket, nil
	}
	return nil, nil

}
