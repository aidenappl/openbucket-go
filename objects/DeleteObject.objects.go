package objects

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
)

// DeleteObject removes one key from a bucket: its file and its metadata row.
//
// Idempotent, like S3: a key whose file is already gone still has its row
// removed, and a key that doesn't exist at all succeeds.
//
// ⚠️ A folder-marker key ("a/") deletes only the marker, never the keys under
// it. It used to os.RemoveAll the directory: every child's file vanished while
// its row stayed, and the rest of a multi-object delete then failed on "no
// such file" — so folder deletes answered 200 and left the contents listed
// (openbucket-go#20, #21).
func DeleteObject(bucket types.Bucket, objectName string) error {
	// Validate path to prevent traversal
	resolvedPath, err := util.ValidateBucketPath(bucket.Name, objectName)
	if err != nil {
		return err
	}

	if strings.HasSuffix(objectName, "/") {
		// The directory goes only once it's empty; children keep it otherwise.
		if err := os.Remove(resolvedPath); err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) && !errors.Is(err, syscall.EEXIST) {
			return fmt.Errorf("failed to delete object folder: %v", err)
		}
	} else {
		if err := os.Remove(resolvedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to delete object file: %v", err)
		}
	}
	pruneEmptyDirs(filepath.Dir(resolvedPath), util.GetBucketPath(bucket.Name))

	// Tags hang off the row; a key with no row has none.
	object, err := GetObject(bucket.Name, objectName, nil)
	if err != nil {
		return fmt.Errorf("failed to get object: %v", err)
	}
	if object != nil {
		if err := DeleteAllObjectTags(bucket.ID, object.ID); err != nil {
			return fmt.Errorf("failed to delete object tags: %v", err)
		}
	}

	// Remove object from the database
	_, err = db.Psql.
		Delete("objects").
		Where(sq.Eq{"objects.bucket_id": bucket.ID, "objects.key": objectName}).
		Exec()
	if err != nil {
		return fmt.Errorf("failed to delete object from database: %v", err)
	}

	return nil // Return nil if deletion is successful
}

// pruneEmptyDirs removes dir and its parents while they are empty, stopping
// at (and keeping) root. Directories are an artefact of storing keys as files:
// deleting a folder's keys in any order otherwise left empty directories, and
// a directory blocks a later upload of a file at that path.
func pruneEmptyDirs(dir, root string) {
	root = filepath.Clean(root)
	for dir = filepath.Clean(dir); dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
		if err := os.Remove(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return // not empty: nor are its parents
		}
	}
}

// DeleteObject removes an object from a bucket
func DeleteVersionedObject(bucket types.Bucket, objectName string, versionId string) error {
	return DeleteObject(bucket, objectName) // For simplicity, we treat versioned objects the same as regular objects TODO: implement versioning
}
