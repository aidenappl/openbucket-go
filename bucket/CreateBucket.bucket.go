package bucket

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/types"
)

func CreateBucket(bucketName string, owner types.UserObject) error {
	filePath := filepath.Join("buckets", bucketName)

	// Does it already exist?
	if fi, err := os.Stat(filePath); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", filePath)
		}
		log.Println("Bucket already exists:", bucketName)
		return fmt.Errorf("bucket already exists: %s", bucketName)
	} else if !errors.Is(err, os.ErrNotExist) {
		// real error
		return fmt.Errorf("stat %s: %w", filePath, err)
	}

	// Parent(s) + bucket
	if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
		return fmt.Errorf("create bucket %s: %w", bucketName, err)
	}

	log.Println("Created bucket:", bucketName)

	// Create base permissions file
	err := SaveBucketPermissions(bucketName, &types.Bucket{
		Name:         bucketName,
		Owner:        owner,
		ACL:          types.BUCKET_ACLPrivate, // Default ACL for new buckets
		Grants:       []types.Grant{},
		CreationDate: types.IsoTime(time.Now()),
	})
	if err != nil {
		log.Println("Error saving permissions file:", err)
		return fmt.Errorf("error saving permissions file: %v", err)
	}

	// Grant creator permissions
	grant := auth.NewGrant(owner.ID, owner.DisplayName, types.FULL_CONTROL)
	err = auth.SaveNewGrant(bucketName, &grant)
	if err != nil {
		log.Println("Error granting creator permissions:", err)
		return fmt.Errorf("error granting creator permissions: %v", err)
	}

	// Create blank tags file
	err = SaveBucketTags(bucketName, nil)
	if err != nil {
		log.Println("Error creating tags file:", err)
		return fmt.Errorf("error creating tags file: %v", err)
	}

	return nil
}
