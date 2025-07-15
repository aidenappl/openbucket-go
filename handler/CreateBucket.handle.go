package handler

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

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
	_, err = permissionsFile.WriteString("<permissions>\n  <read>public</read>\n  <write>private</write>\n</permissions>\n")
	if err != nil {
		log.Println("Error writing to permissions file:", err)
		return fmt.Errorf("error writing to permissions file: %v", err)
	}

	// Respond with 201 Created
	return nil
}
