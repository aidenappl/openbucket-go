package util

import "os"

func BucketExists(bucketName string) bool {
	// validate bucket name
	if bucketName == "" {
		return false
	}

	// Structure bucket
	filePath := "buckets/" + bucketName

	// lookup bucket
	_, err := os.Open(filePath)
	return err == nil
}
