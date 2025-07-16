package auth

import (
	"encoding/xml"
	"fmt"
	"os"

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
