package auth

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/aidenappl/openbucket-go/types"
)

// LoadAuthorizations loads the global authorizations XML file
func LoadAuthorizations() (*types.Authorizations, error) {
	file, err := os.Open("authorizations.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to open authorizations file: %v", err)
	}
	defer file.Close()

	var authorizations types.Authorizations
	decoder := xml.NewDecoder(file)
	err = decoder.Decode(&authorizations)
	if err != nil {
		return nil, fmt.Errorf("failed to decode authorizations XML: %v", err)
	}

	return &authorizations, nil
}

// CheckUserPermissions checks if the user (keyID) has permission to access the given bucket
func CheckUserPermissions(keyID, bucketName string) (*types.Authorization, error) {
	// Load global authorizations to validate the KEY_ID
	authorizations, err := LoadAuthorizations()
	if err != nil {
		return nil, err
	}

	var authorization *types.Authorization
	// Check if the KEY_ID exists in the authorizations list
	for _, auth := range authorizations.Authorizations {
		if auth.KeyID == keyID {
			authorization = &auth
			break
		}
	}

	if authorization == nil {
		return nil, fmt.Errorf("user with KEY_ID %s not found in authorizations", keyID)
	}

	// Load the bucket's permissions
	permissions, err := LoadPermissions(bucketName)
	if err != nil {
		return nil, err
	}

	// Check if the user is granted access to the bucket
	for _, grant := range permissions.Grants {
		if grant.KeyID == keyID {
			return authorization, nil
		}
	}

	return nil, fmt.Errorf("user with KEY_ID %s does not have permission for bucket %s", keyID, bucketName)
}
