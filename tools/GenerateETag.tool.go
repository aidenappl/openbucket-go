package tools

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func GenerateETag(filePath string) (string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()

	_, err = hash.Write([]byte(filePath))
	if err != nil {
		return "", fmt.Errorf("error writing file path to hash: %w", err)
	}

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("error calculating hash: %w", err)
	}

	etag := hex.EncodeToString(hash.Sum(nil))
	return etag, nil
}
