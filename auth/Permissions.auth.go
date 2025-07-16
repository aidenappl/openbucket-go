package auth

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

// LoadPermissions loads the permissions XML file for a specific bucket
func LoadPermissions(bucketName string) (*types.Permissions, error) {
	permissionsFile := fmt.Sprintf("buckets/%s.obpermissions", bucketName)
	file, err := os.Open(permissionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open permissions file: %v", err)
	}
	defer file.Close()

	var permissions types.Permissions
	decoder := xml.NewDecoder(file)
	err = decoder.Decode(&permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode permissions XML: %v", err)
	}

	return &permissions, nil
}

// NewGrant creates a new grant for a specific keyID
func NewGrant(keyID string) types.Grant {
	return types.Grant{
		KeyID:     keyID,
		DateAdded: time.Now(),
	}
}

// UpdatePermissions updates the permissions XML file for a specific bucket
func UpdatePermissions(bucketName string, permissions *types.Permissions) error {
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
