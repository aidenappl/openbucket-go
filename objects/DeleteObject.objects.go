package objects

import "os"

// DeleteObject removes an object from a bucket
func DeleteObject(bucketName, objectName string) error {
	// Structure bucket
	filePath := "buckets/" + bucketName

	// Attempt to remove the object file
	err := os.Remove(filePath + "/" + objectName)
	if err != nil {
		return err // Return the error if deletion fails
	}

	// Remove the associated metadata file
	err = os.Remove(filePath + "/" + objectName + ".obmeta")
	if err != nil {
		return err
	}

	return nil // Return nil if deletion is successful
}

// DeleteObject removes an object from a bucket
func DeleteVersionedObject(bucketName, objectName string, versionId string) error {
	return DeleteObject(bucketName, objectName) // For simplicity, we treat versioned objects the same as regular objects TODO: implement versioning
}
