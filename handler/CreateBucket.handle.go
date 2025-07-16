package handler

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/types"
)

func CreateBucket(bucket string) error {

	filePath := filepath.Join("buckets", bucket)

	_, err := os.Stat(filePath)
	if os.IsExist(err) {

		log.Println("Bucket already exists:", bucket)
		return fmt.Errorf("bucket already exists: %s", bucket)
	} else if os.IsNotExist(err) {

		log.Println("Creating bucket:", bucket)

		if err := os.MkdirAll(filepath.Join("buckets"), os.ModePerm); err != nil {
			log.Println("Error creating buckets directory:", err)
			return fmt.Errorf("error creating buckets directory: %v", err)
		}

		err = os.Mkdir(filePath, os.ModePerm)
		if err != nil {
			log.Println("Error creating bucket directory:", err)
			return fmt.Errorf("failed to create bucket: %v", err)
		}
	} else if err != nil {

		log.Println("Error accessing file:", err)
		return fmt.Errorf("error accessing file: %v", err)
	}

	permissionsFile, err := os.Create(filePath + ".obpermissions")
	if err != nil {
		log.Println("Error creating permissions file:", err)
		return fmt.Errorf("error creating permissions file: %v", err)
	}

	defer permissionsFile.Close()

	permissions := types.Permissions{
		AllowGlobalRead:  false,
		AllowGlobalWrite: false,
		Grants:           []types.Grant{},
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

	return nil
}
