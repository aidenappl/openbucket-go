package util

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateBucketPath validates that a key does not escape the bucket directory
// and returns the resolved file path. Returns an error if path traversal is detected.
func ValidateBucketPath(bucketName, key string) (string, error) {
	if key == "" || key == "." || key == ".." || strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid key: path traversal detected")
	}
	base := filepath.Join("buckets", bucketName)
	full := filepath.Join(base, key)
	absBase, _ := filepath.Abs(base)
	absFull, _ := filepath.Abs(full)
	if !strings.HasPrefix(absFull, absBase+string(filepath.Separator)) && absFull != absBase {
		return "", fmt.Errorf("invalid key: path traversal detected")
	}
	return full, nil
}
