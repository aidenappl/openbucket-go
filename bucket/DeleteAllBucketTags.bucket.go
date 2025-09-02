package bucket

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

func DeleteAllBucketTags(bucketID int) error {
	_, err := db.Psql.Delete("bucket_tags").
		Where(sq.Eq{"bucket_id": bucketID}).
		Exec()
	return err
}
