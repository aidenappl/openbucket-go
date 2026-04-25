package bucket

import (
	"fmt"
	"log"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// bucketCacheEntry holds a cached bucket with expiry time
type bucketCacheEntry struct {
	bucket    *types.Bucket
	expiresAt time.Time
}

// bucketCache is a simple in-memory cache for bucket lookups
var bucketCache sync.Map

// bucketCacheTTL is how long cached bucket entries are valid
const bucketCacheTTL = 2 * time.Minute

// getCachedBucket retrieves a bucket from cache if valid
func getCachedBucket(name string) (*types.Bucket, bool) {
	if entry, ok := bucketCache.Load(name); ok {
		cached := entry.(*bucketCacheEntry)
		if time.Now().Before(cached.expiresAt) {
			return cached.bucket, true
		}
		bucketCache.Delete(name)
	}
	return nil, false
}

// setCachedBucket stores a bucket in cache
func setCachedBucket(name string, b *types.Bucket) {
	bucketCache.Store(name, &bucketCacheEntry{
		bucket:    b,
		expiresAt: time.Now().Add(bucketCacheTTL),
	})
}

// InvalidateBucketCache removes a specific bucket from the cache
func InvalidateBucketCache(name string) {
	bucketCache.Delete(name)
}

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

	// Check cache first (only for default options with grants)
	if opt.IncludeGrants {
		if cached, ok := getCachedBucket(bucketName); ok {
			return cached, nil
		}
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

			// Cache the complete bucket (with grants)
			setCachedBucket(bucketName, &bucket)
		}

		return &bucket, nil
	}
	return nil, nil
}
