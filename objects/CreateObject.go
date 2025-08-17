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

func CreateObject(filePath string, key string, bucket string, bodyContent io.Reader, osContent *[]byte, user *types.UserObject) (*string, error) {
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
	metadataFile, err := os.Create(metadataFilePath)
	if err != nil {
		log.Println("Error saving metadata:", err)
		return nil, err
	}
	defer metadataFile.Close()

	metadataXML, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		log.Println("Error marshalling metadata to XML:", err)
		return nil, err
	}

	_, err = metadataFile.WriteString(string(metadataXML))
	if err != nil {
		return nil, err
	}

	return &etag, nil
}
