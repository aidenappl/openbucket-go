package auth

import (
	"github.com/aidenappl/openbucket-go/types"
)

type PermissionType string

const (
	ReadPermission  PermissionType = "read"
	WritePermission PermissionType = "write"
	AdminPermission PermissionType = "admin"
)

func UserHasPermissionToObject(session *types.Authorization, grant *types.Grant, objectMetadata *types.ObjectMetadata, bucket string, object string, requiredPermissionType PermissionType) bool {
	// Validate object metadata
	if objectMetadata == nil {
		return false
	}

	// validate object permissions in objectMetadata
	if objectMetadata.Owner.ID == session.KeyID {
		return true
	}

	// check if object is private
	// TODO: validate object ACLs (has not been implemented yet)

	// Check user's bucket permissions
	if requiredPermissionType == ReadPermission {
		if types.IsReadPermission(grant.Permission) {
			return true
		}
	}

	if requiredPermissionType == WritePermission {
		if types.IsWritePermission(grant.Permission) {
			return true
		}
	}

	if requiredPermissionType == AdminPermission {
		if grant.Permission == types.FULL_CONTROL {
			return true
		}
	}

	return false
}
