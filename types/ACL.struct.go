package types

// ACL defines the structure of an Access Control List (ACL) in OpenBucket.
type Permission string

// Permission constants for bucket and object permissions.
const (
	READ         Permission = "READ"
	WRITE        Permission = "WRITE"
	READ_ACP     Permission = "READ_ACP"
	WRITE_ACP    Permission = "WRITE_ACP"
	FULL_CONTROL Permission = "FULL_CONTROL"
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
