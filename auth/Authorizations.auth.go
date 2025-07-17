package auth

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/aidenappl/openbucket-go/types"
)

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

func CheckUserExists(keyID string) (*types.Authorization, error) {
	authorizations, err := LoadAuthorizations()
	if err != nil {
		return nil, fmt.Errorf("failed to load authorizations: %v", err)
	}

	for _, auth := range authorizations.Authorizations {
		if auth.KeyID == keyID {
			return &auth, nil
		}
	}

	return nil, nil
}

func CheckUserPermissions(keyID, bucketName string) (*types.Grant, error) {

	authorizations, err := LoadAuthorizations()
	if err != nil {
		return nil, err
	}

	var authorization *types.Authorization

	for _, auth := range authorizations.Authorizations {
		if auth.KeyID == keyID {
			authorization = &auth
			break
		}
	}

	if authorization == nil {
		return nil, fmt.Errorf("user with KEY_ID %s not found in authorizations", keyID)
	}

	permissions, err := LoadPermissions(bucketName)
	if err != nil {
		return nil, err
	}

	for _, grant := range permissions.Grants {
		if grant.KeyID == keyID {
			return &grant, nil
		}
	}

	return nil, nil
}
