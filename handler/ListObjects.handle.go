package handler

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/metadata"
	"github.com/aidenappl/openbucket-go/types"
)

func ListObjects(bucket string) (*[]types.ObjectContent, error) {
	// Define the path to the bucket's objects (files)
	bucketDir := filepath.Join("buckets", bucket)

	// Check if the bucket directory exists
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("bucket does not exist: %s", bucket)
	} else if err != nil {
		return nil, fmt.Errorf("error accessing bucket directory: %v", err)
	}

	files, err := os.ReadDir(bucketDir)
	if err != nil {
		return nil, fmt.Errorf("error reading bucket directory: %v", err)
	}

	// Create an ObjectList to hold the response data
	var objectList []types.ObjectContent

	// Iterate over the files and add them to the object list
	for _, file := range files {
		// Skip directories and metadata files (.obmeta)
		if file.IsDir() || strings.HasSuffix(file.Name(), ".obmeta") {
			continue
		}

		// Check if the .obmeta file exists
		metadataFilePath := filepath.Join(bucketDir, file.Name()+".obmeta")
		if _, err := os.Stat(metadataFilePath); err == nil {
			// If .obmeta exists, read and parse the metadata XML
			metadataFile, err := os.Open(metadataFilePath)
			if err != nil {
				return nil, fmt.Errorf("error opening .obmeta file: %v", err)
			}
			defer metadataFile.Close()

			// Parse the XML metadata
			var metadata metadata.Metadata
			decoder := xml.NewDecoder(metadataFile)
			if err := decoder.Decode(&metadata); err != nil {
				return nil, fmt.Errorf("error decoding .obmeta XML: %v", err)
			}

			// Get file info
			info, err := file.Info()
			if err != nil {
				return nil, fmt.Errorf("error getting file info: %v", err)
			}

			// Append file info to object list
			objectList = append(objectList, types.ObjectContent{
				Key:          file.Name(),
				LastModified: metadata.LastModified,
				CreatedAt:    metadata.UploadedAt,
				ETag:         metadata.ETag,
				Size:         info.Size(),
			})
		} else {
			// If .obmeta file is missing, continue to next file
			continue
		}
	}
	// Return the object list
	return &objectList, nil
}

func ListObjectsXML(bucket string) (*types.ObjectList, error) {
	objectList, err := ListObjects(bucket)
	if err != nil {
		return nil, err
	}

	// Create an ObjectList to hold the response data
	objectListXML := &types.ObjectList{
		XMLName:  xml.Name{Local: "ListBucketResult"},
		Contents: *objectList,
	}

	return objectListXML, nil
}
