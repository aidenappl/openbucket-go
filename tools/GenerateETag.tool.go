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

// ⚠️ An ETag is the hex MD5 of the object's bytes and nothing else, as in S3
// for a single-part PUT. Clients verify uploads and copies against it. It
// used to be md5(filePath + content), so rclone flagged every object copied
// into cdn.hillview.tv as corrupted (2026-10-18). Two keys holding the same
// bytes share an ETag; nothing here may treat an ETag as unique.

// ETagWriter wraps a writer and calculates the ETag (MD5) while writing
type ETagWriter struct {
	writer io.Writer
	hash   hash.Hash
}

// NewETagWriter creates a writer that calculates the ETag while writing
func NewETagWriter(w io.Writer) *ETagWriter {
	return &ETagWriter{writer: w, hash: md5.New()}
}

// Write writes data to the underlying writer and updates the hash
func (e *ETagWriter) Write(p []byte) (n int, err error) {
	n, err = e.writer.Write(p)
	if n > 0 {
		e.hash.Write(p[:n])
	}
	return n, err
}

// Sum returns the raw MD5 of everything written
func (e *ETagWriter) Sum() []byte {
	return e.hash.Sum(nil)
}

// ETag returns the calculated ETag
func (e *ETagWriter) ETag() types.ETag {
	return types.ETag(hex.EncodeToString(e.Sum()))
}

// ETagOf returns the ETag of a byte slice
func ETagOf(b []byte) types.ETag {
	sum := md5.Sum(b)
	return types.ETag(hex.EncodeToString(sum[:]))
}

// GenerateETag generates the ETag of a file on disk. A directory (folder
// marker) has the ETag of zero bytes.
func GenerateETag(filePath string) (types.ETag, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to get file info: %w", err)
	}

	hash := md5.New()
	if !fileInfo.IsDir() {
		if _, err := io.Copy(hash, file); err != nil {
			return "", fmt.Errorf("error calculating hash: %w", err)
		}
	}
	return types.ETag(hex.EncodeToString(hash.Sum(nil))), nil
}
