package objects

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
)

// ErrBadDigest means the body did not match the client's Content-MD5. The
// existing object, if any, is left untouched.
var ErrBadDigest = errors.New("the Content-MD5 you specified did not match what we received")

func CreateObject(filePath string, key string, bucket types.Bucket, bodyContent io.Reader, osContent *[]byte, user *types.Authorization, public *bool) (*types.ETag, error) {
	return CreateObjectVerified(filePath, key, bucket, bodyContent, osContent, user, public, nil)
}

// CreateObjectVerified is CreateObject that, when wantMD5 is non-nil, refuses
// a body whose MD5 differs with ErrBadDigest. The body is written to a temp
// file and renamed into place, so a refused or failed upload never replaces
// the object already stored under the key.
func CreateObjectVerified(filePath string, key string, bucket types.Bucket, bodyContent io.Reader, osContent *[]byte, user *types.Authorization, public *bool, wantMD5 []byte) (*types.ETag, error) {
	if public == nil {
		public = new(bool)
		*public = false
	}

	// Validate path to prevent traversal
	if strings.Contains(key, "..") {
		return nil, fmt.Errorf("invalid key: path traversal detected")
	}
	base := filepath.Join("buckets", bucket.Name)
	absBase, _ := filepath.Abs(base)
	absFull, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFull, absBase+string(filepath.Separator)) && absFull != absBase {
		return nil, fmt.Errorf("invalid key: path traversal detected")
	}

	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		log.Println("Error creating directory:", err)
		return nil, err
	}

	file, err := os.CreateTemp(filepath.Dir(filePath), ".upload-*")
	if err != nil {
		log.Println("Error creating file:", err)
		return nil, err
	}
	tmpPath := file.Name()
	renamed := false
	defer func() {
		file.Close()
		if !renamed {
			os.Remove(tmpPath)
		}
	}()

	// Create ETag writer to calculate hash while writing (avoids re-reading file)
	etagWriter := tools.NewETagWriter(file)

	if bodyContent != nil {
		_, err = io.Copy(etagWriter, bodyContent)
		if err != nil {
			log.Println("Error saving file:", err)
			return nil, err
		}
	} else if osContent != nil {
		_, err = etagWriter.Write(*osContent)
		if err != nil {
			log.Println("Error saving file:", err)
			return nil, err
		}
	}

	if wantMD5 != nil && !bytes.Equal(wantMD5, etagWriter.Sum()) {
		return nil, ErrBadDigest
	}

	// Get ETag from writer (no need to re-read file)
	etag := etagWriter.ETag()

	stat, err := file.Stat()
	if err != nil {
		log.Println("Error getting file info:", err)
		return nil, err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		log.Println("Error moving file into place:", err)
		return nil, err
	}
	renamed = true

	_, err = db.Psql.Insert("objects").Columns(
		"bucket_id",
		"etag",
		"key",
		"owner_id",
		"public",
		"last_modified",
		"uploaded_at",
		"version_id",
		"size",
	).Values(
		bucket.ID,
		etag,
		key,
		user.ID,
		public,
		time.Now(),
		time.Now(),
		1,
		stat.Size(),
	).Suffix(objectUpsert).Exec()
	if err != nil {
		log.Println("Error inserting object metadata:", err)
		return nil, err
	}

	return &etag, nil
}
