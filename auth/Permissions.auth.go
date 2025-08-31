package auth

import (
	"fmt"
	"time"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

const (
	s3NS  = "http://s3.amazonaws.com/doc/2006-03-01/"
	xsiNS = "http://www.w3.org/2001/XMLSchema-instance"
)

func LoadBucketPermissions(bucketName string) (*types.Bucket, error) {
	// TODO: REPAIR
	return nil, nil
}

type SaveNewGrantReq struct {
	BucketID   int
	GranteeID  int
	Permission types.Permission
}

func SaveNewGrant(req SaveNewGrantReq) error {
	query := db.Psql.Insert("bucket_permissions").Columns(
		"bucket_id",
		"grantee_id",
		"permission",
		"date_added",
	).Values(
		req.BucketID,
		req.GranteeID,
		req.Permission,
		time.Now(),
	)

	_, err := query.Exec()
	if err != nil {
		return fmt.Errorf("failed to save new grant: %v", err)
	}

	return nil
}

func UpdateGrant(bucketName string, grant *types.Grant) error {
	// TODO: Repair
	return nil
}

func UpdateBucketPermissions(bucket string, metadata *types.Bucket) error {
	// TODO: Repair
	return nil
}
