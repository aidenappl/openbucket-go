package bucket

import (
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

type NewGrantReq struct {
	BucketID   int
	GranteeID  int
	Permission types.Permission
}

func NewGrant(req NewGrantReq) error {
	_, err := db.Psql.
		Insert("bucket_permissions").
		Columns("bucket_id", "grantee_id", "permission").
		Values(req.BucketID, req.GranteeID, req.Permission).
		Exec()

	if err != nil {
		return err
	}

	return nil
}
