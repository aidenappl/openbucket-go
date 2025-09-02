package objects

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

func ObjectExists(bucketName string, objectName string) bool {
	query := db.Psql.
		Select("objects.id").
		From("objects").
		Join("buckets ON objects.bucket_id = buckets.id").
		Where(sq.Eq{"buckets.name": bucketName, "objects.key": objectName})
	var id int
	err := query.Scan(&id)
	return err == nil
}
