package util

import "strings"

func GetBucketPath(bucketName string) string {
	if bucketName == "" || strings.Contains(bucketName, "..") || strings.Contains(bucketName, "/") {
		return ""
	}
	return "buckets/" + bucketName
}
