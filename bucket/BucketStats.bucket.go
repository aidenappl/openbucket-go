package bucket

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

type BucketStatsResult struct {
	ObjectCount int64 `json:"object_count"`
	TotalSize   int64 `json:"total_size"`
}

// GetBucketStats returns the object count and total size for a bucket.
func GetBucketStats(bucketName string) (*BucketStatsResult, error) {
	row := db.Psql.
		Select("COUNT(*)", "COALESCE(SUM(objects.size), 0)").
		From("objects").
		Join("buckets ON objects.bucket_id = buckets.id").
		Where(sq.Eq{"buckets.name": bucketName}).
		QueryRow()

	var stats BucketStatsResult
	if err := row.Scan(&stats.ObjectCount, &stats.TotalSize); err != nil {
		return nil, fmt.Errorf("query bucket stats: %w", err)
	}
	return &stats, nil
}
