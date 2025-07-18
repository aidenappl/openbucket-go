package handler

import (
	"fmt"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/types"
)

func GrantAccess(bucketName string, keyID string, acl string) error {
	permissions, err := auth.LoadBucketPermissions(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions for bucket %s: %v", bucketName, err)
	}

	for _, grant := range permissions.Grants {
		if grant.Grantee.ID == keyID {
			return fmt.Errorf("keyID %s already has access to bucket %s", keyID, bucketName)
		}
	}

	if acl == "" {
		acl = "READ"
	}

	authr, err := auth.CheckUserExists(keyID)
	if err != nil {
		return fmt.Errorf("failed to load authorizations: %v", err)
	}

	if authr == nil {
		return fmt.Errorf("keyID %s is not valid", keyID)
	}

	grantType := types.ConvertToPermission(acl)
	if grantType == types.ACLUnknown {
		return fmt.Errorf("invalid ACL type: %s", acl)
	}

	permissions.Grants = append(permissions.Grants, auth.NewGrant(keyID, authr.Name, grantType))

	err = auth.UpdateBucketPermissions(bucketName, permissions)
	if err != nil {
		return fmt.Errorf("failed to save permissions for bucket %s: %v", bucketName, err)
	}

	return nil

}
