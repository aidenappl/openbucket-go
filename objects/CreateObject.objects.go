package objects

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
)

func CreateObject(filePath string, key string, bucket types.Bucket, bodyContent io.Reader, osContent *[]byte, user *types.Authorization) (*types.ETag, error) {
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

	stat, err := file.Stat()
	if err != nil {
		log.Println("Error getting file info:", err)
		return nil, err
	}

	if bodyContent != nil {
		_, err = io.Copy(file, bodyContent)
		if err != nil {
			log.Println("Error saving file:", err)
			return nil, err
		}
	} else if osContent != nil {
		_, err = file.Write(*osContent)
		if err != nil {
			log.Println("Error saving file:", err)
			return nil, err
		}
	}

	etag, err := tools.GenerateETag(filePath)
	if err != nil {
		log.Println("Error generating ETag:", err)
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
		false,
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
