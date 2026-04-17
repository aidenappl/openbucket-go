package objects

import (
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

func CreateObject(filePath string, key string, bucket types.Bucket, bodyContent io.Reader, osContent *[]byte, user *types.Authorization, public *bool) (*types.ETag, error) {
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

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("Error creating file:", err)
		return nil, err
	}
	defer file.Close()

	// Create ETag writer to calculate hash while writing (avoids re-reading file)
	etagWriter := tools.NewETagWriter(file, filePath)

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

	// Get ETag from writer (no need to re-read file)
	etag := etagWriter.ETag()

	stat, err := file.Stat()
	if err != nil {
		log.Println("Error getting file info:", err)
		return nil, err
	}

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
	).Exec()
	if err != nil {
		log.Println("Error inserting object metadata:", err)
		return nil, err
	}

	return &etag, nil
}
