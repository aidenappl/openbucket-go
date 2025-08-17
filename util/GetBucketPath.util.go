package util

func GetBucketPath(bucketName string) string {
	// validate bucket name
	if bucketName == "" {
		return ""
	}

	// Structure bucket
	return "buckets/" + bucketName
}

func GetBucketMetadataPath(bucketName string) string {
	// validate bucket name
	if bucketName == "" {
		return ""
	}

	// Structure metadata file path
	return "buckets/" + bucketName + ".obpermissions"
}
