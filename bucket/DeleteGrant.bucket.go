package bucket

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

// DeleteGrant removes a single grant by ID.
func DeleteGrant(grantID int) error {
	res, err := db.Psql.
		Delete("bucket_permissions").
		Where(sq.Eq{"id": grantID}).
		Exec()
	if err != nil {
		return fmt.Errorf("delete grant: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("grant not found")
	}
	return nil
}
