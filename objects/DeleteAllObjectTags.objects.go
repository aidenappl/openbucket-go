package objects

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

func DeleteAllObjectTags(bucketID int, objectID int) error {
	_, err := db.Psql.Delete("object_tags").
		Where(sq.Eq{"bucket_id": bucketID, "object_id": objectID}).
		Exec()
	return err
}
