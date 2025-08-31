package objects

import (
	"encoding/xml"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
)

func CreateObject(filePath string, key string, bucket string, bodyContent io.Reader, osContent *[]byte, user *types.UserObject) (*types.ETag, error) {
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

	metadata := &types.ObjectMetadata{
		Xmlns:        "http://s3.amazonaws.com/doc/2006-03-01/",
		ETag:         etag,
		Key:          key,
		Bucket:       bucket,
		Owner:        *user,
		Public:       false,
		LastModified: types.IsoTime(time.Now()),
		UploadedAt:   types.IsoTime(time.Now()),
		VersionId:    "1",
		Size:         stat.Size(),
	}

	metadataFilePath := filePath + ".obmeta"
	tmp := metadataFilePath + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return nil, err
	}

	buf, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}

	buf = append([]byte(xml.Header), buf...)
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return nil, err
	}

	err = os.Rename(tmp, metadataFilePath)
	if err != nil {
		return nil, err
	}

	return &etag, nil
}
