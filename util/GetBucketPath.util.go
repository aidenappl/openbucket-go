package util

func GetBucketPath(bucketName string) string {
	// validate bucket name
	if bucketName == "" {
		return ""
	}

	// Structure bucket
	return "buckets/" + bucketName
}
