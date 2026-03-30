package tools

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/aidenappl/openbucket-go/types"
)

// ETagWriter wraps a writer and calculates ETag (MD5) while writing
type ETagWriter struct {
	writer   io.Writer
	hash     hash.Hash
	filePath string
}

// NewETagWriter creates a writer that calculates ETag while writing
func NewETagWriter(w io.Writer, filePath string) *ETagWriter {
	h := md5.New()
	// Include file path in hash for consistency with GenerateETag
	h.Write([]byte(filePath))
	return &ETagWriter{
		writer:   w,
		hash:     h,
		filePath: filePath,
	}
}

// Write writes data to the underlying writer and updates the hash
func (e *ETagWriter) Write(p []byte) (n int, err error) {
	n, err = e.writer.Write(p)
	if n > 0 {
		e.hash.Write(p[:n])
	}
	return n, err
}

// ETag returns the calculated ETag
func (e *ETagWriter) ETag() types.ETag {
	return types.ETag(hex.EncodeToString(e.hash.Sum(nil)))
}

// GenerateETag generates an ETag for an existing file (for backward compatibility)
func GenerateETag(filePath string) (types.ETag, error) {

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

	// check if file is not a directory
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to get file info: %w", err)
	}

	if !fileInfo.IsDir() {
		_, err = io.Copy(hash, file)
		if err != nil {
			return "", fmt.Errorf("error calculating hash: %w", err)
		}
	}

	// fallback to hashing the file path string
	etag := hex.EncodeToString(hash.Sum(nil))
	return types.ETag(etag), nil
}
