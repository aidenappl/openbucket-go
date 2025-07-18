package handler

import (
	"fmt"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/types"
)

func UpdateGrantAccess(bucketName string, keyID string, acl types.Permission) error {
	if acl == types.ACLUnknown || keyID == "" || bucketName == "" {
		return fmt.Errorf("bucket name, key ID, and ACL must be provided")
	}

	permissions, err := auth.LoadBucketPermissions(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions for bucket %s: %v", bucketName, err)
	}

	var found bool
	for _, grant := range permissions.Grants {
		if grant.Grantee.ID == keyID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("keyID %s does not have access to bucket %s", keyID, bucketName)
	}

	authr, err := auth.LoadAuthorizations()
	if err != nil {
		return fmt.Errorf("failed to load authorizations: %v", err)
	}

	valid := false
	for _, cred := range authr.Authorizations {
		if cred.KeyID == keyID {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("keyID %s is not valid", keyID)
	}

	for i, grant := range permissions.Grants {
		if grant.Grantee.ID == keyID {
			permissions.Grants[i].Permission = acl
			break
		}
	}
	if len(permissions.Grants) == 0 {
		return fmt.Errorf("no grants found for keyID %s in bucket %s", keyID, bucketName)
	}

	err = auth.UpdateBucketPermissions(bucketName, permissions)
	if err != nil {
		return fmt.Errorf("failed to save permissions for bucket %s: %v", bucketName, err)
	}

	return nil
}
