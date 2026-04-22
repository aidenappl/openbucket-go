package types

import "time"

// BackupManifest is the top-level structure written as manifest.json inside a
// bucket export archive (.tar.gz).
type BackupManifest struct {
	Version     int                    `json:"version"`
	ExportedAt  time.Time              `json:"exported_at"`
	Bucket      BackupBucket           `json:"bucket"`
	Permissions []BackupPermission     `json:"permissions"`
	Objects     []BackupObjectMetadata `json:"objects"`
}

// BackupBucket captures bucket-level metadata for export.
type BackupBucket struct {
	Name         string      `json:"name"`
	ACL          string      `json:"acl"`
	CreationDate time.Time   `json:"creation_date"`
	Tags         []BackupTag `json:"tags"`
}

// BackupPermission captures a single grant entry, keyed by the grantee's
// access key ID so it can be matched on import without relying on DB IDs.
type BackupPermission struct {
	KeyID      string `json:"key_id"`
	Permission string `json:"permission"`
}

// BackupObjectMetadata captures per-object metadata stored alongside the file
// content in the archive.
type BackupObjectMetadata struct {
	Key          string      `json:"key"`
	ETag         string      `json:"etag"`
	Size         int64       `json:"size"`
	Public       bool        `json:"public"`
	LastModified time.Time   `json:"last_modified"`
	OwnerKeyID   string      `json:"owner_key_id"`
	Tags         []BackupTag `json:"tags"`
}

// BackupTag is a simple key-value pair used for both bucket and object tags.
type BackupTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
