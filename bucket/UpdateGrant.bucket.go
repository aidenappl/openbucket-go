package bucket

import (
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

type UpdateGrantReq struct {
	BucketID   int
	GranteeID  int
	Permission types.Permission
}

func UpdateGrant(req UpdateGrantReq) error {
	_, err := db.Psql.Update("bucket_permissions").
		Set("permission", req.Permission).
		Where("bucket_id = ? AND grantee_id = ?", req.BucketID, req.GranteeID).Exec()
	if err != nil {
		return err
	}
	return nil
}
