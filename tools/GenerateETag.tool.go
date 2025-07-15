package tools

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// GenerateETag generates an ETag based on both the file name and file content.
func GenerateETag(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	// Create an MD5 hash instance
	hash := md5.New()

	// Include the file name in the hash calculation
	// This makes sure that even if the content is the same, a different filename will result in a different ETag
	_, err = hash.Write([]byte(filePath))
	if err != nil {
		return "", fmt.Errorf("error writing file path to hash: %w", err)
	}

	// Copy the file content into the hash
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("error calculating hash: %w", err)
	}

	// Generate the ETag
	etag := hex.EncodeToString(hash.Sum(nil))
	return etag, nil
}
