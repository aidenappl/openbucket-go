package handler

import (
	"fmt"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/types"
)

func GrantAccess(bucketName string, keyID string, acl string) error {
	authr, err := auth.CheckUserExists(keyID)
	if err != nil {
		return fmt.Errorf("failed to load authorizations: %v", err)
	}

	if authr == nil {
		return fmt.Errorf("keyID %s is not valid", keyID)
	}

	b, err := bucket.GetBucket(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions for bucket %s: %v", bucketName, err)
	}

	for _, grant := range b.Grants {
		if grant.Grantee.ID == keyID {
			return fmt.Errorf("keyID %s already has access to bucket %s", keyID, bucketName)
		}
	}

	if acl == "" {
		acl = "READ"
	}

	grantType := types.ConvertToPermission(acl)
	if grantType == types.ACLUnknown {
		return fmt.Errorf("invalid ACL type: %s", acl)
	}

	err = bucket.NewGrant(bucket.NewGrantReq{
		BucketID:   b.ID,
		GranteeID:  authr.ID,
		Permission: grantType,
	})
	if err != nil {
		return fmt.Errorf("failed to save permissions for bucket %s: %v", bucketName, err)
	}

	return nil

}
