package handler

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

func CreateBucket(bucket string, owner types.UserObject) error {

	filePath := filepath.Join("buckets", bucket)

	// Does it already exist?
	if fi, err := os.Stat(filePath); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", filePath)
		}
		log.Println("Bucket already exists:", bucket)
		return fmt.Errorf("bucket already exists: %s", bucket)
	} else if !errors.Is(err, os.ErrNotExist) {
		// real error
		return fmt.Errorf("stat %s: %w", filePath, err)
	}

	// Parent(s) + bucket
	if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
		return fmt.Errorf("create bucket %s: %w", bucket, err)
	}

	log.Println("Created bucket:", bucket)

	permissionsFile, err := os.Create(filePath + ".obpermissions")
	if err != nil {
		log.Println("Error creating permissions file:", err)
		return fmt.Errorf("error creating permissions file: %v", err)
	}

	defer permissionsFile.Close()

	permissions := types.Bucket{
		Name:         bucket,
		Owner:        owner,
		ACL:          types.BUCKET_ACLPrivate, // Default ACL for new buckets
		Grants:       []types.Grant{},
		CreationDate: types.IsoTime(time.Now()),
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
