package types

import "strings"

// ACL defines the structure of an Access Control List (ACL) in OpenBucket.
type Permission string

// Permission constants for object permissions.
const (
	READ         Permission = "READ"
	WRITE        Permission = "WRITE"
	READ_ACP     Permission = "READ_ACP"
	WRITE_ACP    Permission = "WRITE_ACP"
	FULL_CONTROL Permission = "FULL_CONTROL"
	ACLUnknown   Permission = "ACL_UNKNOWN" // Default for unknown ACLs
)

// Bucket ACL Types
const (
	BUCKET_ACLPrivate         Permission = "PRIVATE"           // Only the owner has full access
	BUCKET_ACLPublicRead      Permission = "PUBLIC_READ"       // Public read access, owner has
	BUCKET_ACLPublicWrite     Permission = "PUBLIC_WRITE"      // Public write access, owner has full access
	BUCKET_ACLPublicReadWrite Permission = "PUBLIC_READ_WRITE" // Public read and write access,
)

// Resource distinguishes whether the permission applies to a bucket or an object.
type Resource string

// Resource constants for bucket and object.
const (
	ResourceBucket Resource = "BUCKET"
	ResourceObject Resource = "OBJECT"
)

// PermissionInfo captures the semantics of a permission for a given resource.
type PermissionInfo struct {
	Perm        Permission // e.g. READ
	Resource    Resource   // BUCKET or OBJECT
	Description string     // what the permission allows on that resource
}

// permissionTable is a ready‑to‑use lookup slice for all permissions.
var permissionTable = []PermissionInfo{

	{READ, ResourceBucket, "List the objects in the bucket"},
	{WRITE, ResourceBucket, "Create new objects; owners of existing objects may delete/overwrite them"},
	{READ_ACP, ResourceBucket, "Read the bucket ACL"},
	{WRITE_ACP, ResourceBucket, "Write the bucket ACL"},
	{FULL_CONTROL, ResourceBucket, "READ, WRITE, READ_ACP and WRITE_ACP on the bucket"},

	{READ, ResourceObject, "Read the object data and metadata"},
	{READ_ACP, ResourceObject, "Read the object ACL"},
	{WRITE_ACP, ResourceObject, "Write the object ACL"},
	{FULL_CONTROL, ResourceObject, "READ, READ_ACP and WRITE_ACP on the object"},
}

// Describe returns a description of the permission for the given resource.
func Describe(p Permission, r Resource) (string, bool) {
	for _, row := range permissionTable {
		if row.Perm == p && row.Resource == r {
			return row.Description, true
		}
	}
	return "", false
}

// IsWritePermission checks if the given permission is a write permission.
func IsWritePermission(p Permission) bool {
	return p == WRITE || p == WRITE_ACP || p == FULL_CONTROL
}

// IsReadPermission checks if the given permission is a read permission.
func IsReadPermission(p Permission) bool {
	return p == READ || p == READ_ACP || p == FULL_CONTROL
}

// IsACLModification checks if the given permission is an ACL modification permission.
func IsACLModification(p Permission) bool {
	return p == WRITE_ACP || p == FULL_CONTROL
}

// IsACLReading checks if the given permission is an ACL reading permission.
func IsACLReading(p Permission) bool {
	return p == READ_ACP || p == FULL_CONTROL
}

// AWSHeaderToACL converts an AWS S3 ACL header to a Permission.
func AWSHeaderToACL(header string) Permission {
	switch header {
	case "x-amz-grant-read":
		return READ
	case "x-amz-grant-write":
		return WRITE
	case "x-amz-grant-read-acp":
		return READ_ACP
	case "x-amz-grant-write-acp":
		return WRITE_ACP
	case "x-amz-grant-full-control":
		return FULL_CONTROL
	default:
		return ACLUnknown // Default to FULL_CONTROL if no specific header matches
	}
}

// IsBucketACL checks if the given permission is a bucket ACL.
func IsBucketACL(acl Permission) bool {
	switch acl {
	case BUCKET_ACLPrivate, BUCKET_ACLPublicRead, BUCKET_ACLPublicWrite, BUCKET_ACLPublicReadWrite:
		return true
	default:
		return false
	}
}

// ConvertToBucketACL converts a Permission to a bucket ACL.
func ConvertToBucketACL(acl string) Permission {
	switch acl {
	case "private":
		return BUCKET_ACLPrivate
	case "public-read":
		return BUCKET_ACLPublicRead
	case "public-read-write":
		return BUCKET_ACLPublicReadWrite
	case "public-write":
		return BUCKET_ACLPublicWrite
	default:
		return ACLUnknown // Default to ACLUnknown if no specific ACL matches
	}
}

// ConvertToPermission converts a string to a Permission.
func ConvertToPermission(perm string) Permission {
	perm = strings.ToLower(perm) // Normalize to lowercase
	switch perm {
	case "read":
		return READ
	case "write":
		return WRITE
	case "read-acp":
		return READ_ACP
	case "write-acp":
		return WRITE_ACP
	case "full-control":
		return FULL_CONTROL
	default:
		return ACLUnknown // Default to ACLUnknown if no specific permission matches
	}
}

// func IsBucketACLRead checks if the given permission is a read permission for a bucket ACL.
func IsBucketACLRead(acl Permission) bool {
	return acl == BUCKET_ACLPublicRead || acl == BUCKET_ACLPublicReadWrite
}

// func IsBucketACLWrite checks if the given permission is a write permission for a bucket ACL.
func IsBucketACLWrite(acl Permission) bool {
	return acl == BUCKET_ACLPublicWrite || acl == BUCKET_ACLPublicReadWrite
}
