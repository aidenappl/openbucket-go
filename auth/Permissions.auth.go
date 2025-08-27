package auth

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

const (
	s3NS  = "http://s3.amazonaws.com/doc/2006-03-01/"
	xsiNS = "http://www.w3.org/2001/XMLSchema-instance"
)

func permissionsPath(bucket string) string {
	return filepath.Join("buckets", bucket, ".obpermissions")
}

func LoadBucketPermissions(bucketName string) (*types.Bucket, error) {
	permissionsFile := permissionsPath(bucketName)
	file, err := os.Open(permissionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open permissions file: %v", err)
	}
	defer file.Close()

	var permissions types.Bucket
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&permissions); err != nil {
		return nil, fmt.Errorf("failed to decode permissions XML: %v", err)
	}
	return &permissions, nil
}

func NewGrant(keyID, displayName string, acl types.Permission) types.Grant {
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
	permissions.Grants = append(permissions.Grants, *grant)
	return UpdateBucketPermissions(bucketName, permissions)
}

func UpdateGrant(bucketName string, grant *types.Grant) error {
	permissions, err := LoadBucketPermissions(bucketName)
	if err != nil {
		return fmt.Errorf("failed to load permissions: %v", err)
	}
	for i, g := range permissions.Grants {
		if g.Grantee.ID == grant.Grantee.ID {
			permissions.Grants[i] = *grant
			return UpdateBucketPermissions(bucketName, permissions)
		}
	}
	// If not found, append (optional behavior)
	permissions.Grants = append(permissions.Grants, *grant)
	return UpdateBucketPermissions(bucketName, permissions)
}

func UpdateBucketPermissions(bucket string, metadata *types.Bucket) error {
	// atomic write
	tmp := permissionsPath(bucket) + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return err
	}

	metadata.Xmlns = "http://s3.amazonaws.com/doc/2006-03-01/"

	buf, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	buf = append([]byte(xml.Header), buf...)
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, permissionsPath(bucket))
}
