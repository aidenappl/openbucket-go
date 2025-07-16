package handler

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/types"
)

// Authorization represents each user in the global authorizations file
type Authorization struct {
	Name        string `xml:"Name"`
	KeyID       string `xml:"KEY_ID"`
	SecretKey   string `xml:"SECRET_KEY"`
	DateCreated string `xml:"Date_Created"`
}

// Authorizations represents the structure of the authorizations XML
type Authorizations struct {
	XMLName       xml.Name        `xml:"Authorizations"`
	Authorization []Authorization `xml:"Authorization"`
}

// Permissions represents the structure of the bucket-specific permissions file
type Permissions struct {
	XMLName xml.Name `xml:"permissions"`
	Read    string   `xml:"read"`
	Write   string   `xml:"write"`
	Grants  []string `xml:"grants>grant"`
}

func CreateBucket(bucket string) error {
	// Define the file path for the object in the bucket
	filePath := filepath.Join("buckets", bucket)

	_, err := os.Stat(filePath)
	if os.IsExist(err) {
		// If the file exists, return 409 Conflict
		log.Println("Bucket already exists:", bucket)
		return fmt.Errorf("bucket already exists: %s", bucket)
	} else if os.IsNotExist(err) {
		// If the file does not exist, proceed to create the bucket
		log.Println("Creating bucket:", bucket)

		// Check if the parent directory exists, if not create it
		if err := os.MkdirAll(filepath.Join("buckets"), os.ModePerm); err != nil {
			log.Println("Error creating buckets directory:", err)
			return fmt.Errorf("error creating buckets directory: %v", err)
		}

		// Create the bucket directory
		err = os.Mkdir(filePath, os.ModePerm)
		if err != nil {
			log.Println("Error creating bucket directory:", err)
			return fmt.Errorf("failed to create bucket: %v", err)
		}
	} else if err != nil {
		// If there's any other error, return 500 Internal Server Error
		log.Println("Error accessing file:", err)
		return fmt.Errorf("error accessing file: %v", err)
	}

	// Add permissions xml file
	permissionsFile, err := os.Create(filePath + ".obpermissions")
	if err != nil {
		log.Println("Error creating permissions file:", err)
		return fmt.Errorf("error creating permissions file: %v", err)
	}

	defer permissionsFile.Close()

	// Write default permissions to the file
	permissions := types.Permissions{
		AllowGlobalRead:  false,
		AllowGlobalWrite: false,
		Grants:           []string{},
	}

	permissionsXML, err := xml.MarshalIndent(permissions, "", "  ")
	if err != nil {
		log.Println("Error marshalling permissions to XML:", err)
		return fmt.Errorf("error marshalling permissions to XML: %v", err)
	}

	_, err = permissionsFile.WriteString(string(permissionsXML))
	if err != nil {
		log.Println("Error writing to permissions file:", err)
		return fmt.Errorf("error writing to permissions file: %v", err)
	}

	// Respond with 201 Created
	return nil
}
