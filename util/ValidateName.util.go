package util

import (
	"fmt"
	"strings"
)

// ValidateBucketName checks that a bucket name conforms to S3-compatible naming
// rules and is safe for filesystem use. Returns nil if valid.
func ValidateBucketName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("bucket name must not contain '..'")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.') {
			return fmt.Errorf("bucket name must only contain lowercase letters, numbers, hyphens, and periods")
		}
	}
	if name[0] == '-' || name[0] == '.' || name[len(name)-1] == '-' || name[len(name)-1] == '.' {
		return fmt.Errorf("bucket name must start and end with a letter or number")
	}
	return nil
}

// ValidateObjectKey checks that an object key is safe and within S3 limits.
func ValidateObjectKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("object key must not be empty")
	}
	if len(key) > 1024 {
		return fmt.Errorf("object key must not exceed 1024 characters")
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("object key must not contain '..'")
	}
	if strings.ContainsAny(key, "\x00") {
		return fmt.Errorf("object key must not contain null bytes")
	}
	return nil
}
