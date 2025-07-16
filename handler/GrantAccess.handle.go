package handler

import (
	"fmt"

	"github.com/aidenappl/openbucket-go/auth"
)

func GrantAccess(bucketName string, keyID string) error {
	permissions, err := auth.LoadPermissions(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions for bucket %s: %v", bucketName, err)
	}

	// Check if the keyID already has access
	for _, grant := range permissions.Grants {
		if grant.KeyID == keyID {
			return fmt.Errorf("keyID %s already has access to bucket %s", keyID, bucketName)
		}
	}

	// Check if the keyID is valid
	authr, err := auth.LoadAuthorizations()
	if err != nil {
		return fmt.Errorf("failed to load authorizations: %v", err)
	}

	valid := false
	for _, cred := range authr.Authorizations {
		if cred.KeyID == keyID {
			// KeyID is valid, proceed to grant access
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("keyID %s is not valid", keyID)
	}

	// If the keyID does not have access, add a new grant
	permissions.Grants = append(permissions.Grants, auth.NewGrant(keyID))

	// Save the updated permissions
	err = auth.UpdatePermissions(bucketName, permissions)
	if err != nil {
		return fmt.Errorf("failed to save permissions for bucket %s: %v", bucketName, err)
	}

	return nil

}
