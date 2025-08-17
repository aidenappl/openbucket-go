package util

import "os"

func ObjectExists(bucketName, objectName string) bool {
	if bucketName == "" || objectName == "" {
		return false
	}

	// Structure bucket
	filePath := "buckets/" + bucketName

	// lookup object
	_, err := os.Open(filePath + "/" + objectName)
	return err == nil
}
