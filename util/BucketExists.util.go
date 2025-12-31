package util

import "os"

func BucketExists(bucketName string) bool {
	// validate bucket name
	if bucketName == "" {
		return false
	}

	// Structure bucket
	filePath := "buckets/" + bucketName

	// lookup bucket using Stat to avoid file handle leak
	_, err := os.Stat(filePath)
	return err == nil
}
