package objects

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

func ObjectExists(bucketName string, objectName string) bool {
	var id int
	err := db.Psql.
		Select("objects.id").
		From("objects").
		Join("buckets ON objects.bucket_id = buckets.id").
		Where(sq.Eq{"buckets.name": bucketName, "objects.key": objectName}).
		QueryRow().
		Scan(&id)
	return err == nil
}
