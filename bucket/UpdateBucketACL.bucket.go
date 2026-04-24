package bucket

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// UpdateBucketACL sets the ACL on a bucket by ID.
func UpdateBucketACL(bucketID int, acl types.Permission) error {
	res, err := db.Psql.
		Update("buckets").
		Set("acl", acl).
		Where(sq.Eq{"id": bucketID}).
		Exec()
	if err != nil {
		return fmt.Errorf("update bucket acl: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("bucket not found")
	}
	return nil
}
