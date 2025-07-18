package auth

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

func LoadBucketPermissions(bucketName string) (*types.Bucket, error) {
	permissionsFile := fmt.Sprintf("buckets/%s.obpermissions", bucketName)
	file, err := os.Open(permissionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open permissions file: %v", err)
	}
	defer file.Close()

	var permissions types.Bucket
	decoder := xml.NewDecoder(file)
	err = decoder.Decode(&permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode permissions XML: %v", err)
	}

	return &permissions, nil
}

func NewGrant(keyID string, displayName string, acl types.Permission) types.Grant {
	return types.Grant{
		Permission: acl,
		Grantee: types.Grantee{
			Type:        "CanonicalUser",
			ID:          keyID,
			DisplayName: displayName,
		},
		DateAdded: types.IsoTime(time.Now()),
	}
}

func SaveNewGrant(bucketName string, grant *types.Grant) error {
	permissions, err := LoadBucketPermissions(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions: %v", err)
	}

	// Add the new grant to the permissions
	permissions.Grants = append(permissions.Grants, *grant)

	return UpdateBucketPermissions(bucketName, permissions)
}

func UpdateGrant(bucketName string, grant *types.Grant) error {
	permissions, err := LoadBucketPermissions(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions: %v", err)
	}

	// Update the grant in the permissions
	for i, existingGrant := range permissions.Grants {
		if existingGrant.Grantee.ID == grant.Grantee.ID {
			permissions.Grants[i] = *grant
			break
		}
	}

	return UpdateBucketPermissions(bucketName, permissions)
}

func UpdateBucketPermissions(bucketName string, permissions *types.Bucket) error {
	permissionsFile := fmt.Sprintf("buckets/%s.obpermissions", bucketName)
	file, err := os.Create(permissionsFile)
	if err != nil {
		return fmt.Errorf("failed to create permissions file: %v", err)
	}
	defer file.Close()

	permissionsXML, err := xml.MarshalIndent(permissions, "", "  ")
	if err != nil {
		log.Println("Error marshalling permissions to XML:", err)
		return fmt.Errorf("error marshalling permissions to XML: %v", err)
	}

	_, err = file.WriteString(string(permissionsXML))
	if err != nil {
		log.Println("Error writing to permissions file:", err)
		return fmt.Errorf("error writing to permissions file: %v", err)
	}

	return nil
}
