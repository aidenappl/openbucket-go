package objects

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
)

// Multipart uploads (S3 CreateMultipartUpload / UploadPart /
// CompleteMultipartUpload / AbortMultipartUpload).
//
// State lives on disk, not in Postgres: each upload is a directory
// buckets/.multipart/<uploadId>/ holding upload.json and one file per part
// (part-00001, with its MD5 in part-00001.md5). It sits under buckets/ so it is
// on the same volume as the objects (Complete writes the final file next to
// its key and renames it into place), and no bucket can be named ".multipart"
// (bucket names never start with a dot). Abandoned uploads are removed by
// CleanupStaleMultipartUploads.

const (
	// MultipartRoot holds in-progress uploads.
	MultipartRoot = "buckets/.multipart"

	// MinPartSize is S3's minimum size for every part but the last.
	MinPartSize = 5 << 20
	// MaxPartSize is S3's maximum part size.
	MaxPartSize = 5 << 30
	// MaxPartNumber is S3's highest part number.
	MaxPartNumber = 10000
)

var (
	ErrNoSuchUpload     = errors.New("the specified multipart upload does not exist")
	ErrInvalidPart      = errors.New("one or more of the specified parts could not be found or its ETag did not match")
	ErrInvalidPartOrder = errors.New("the list of parts was not in ascending order")
	ErrEntityTooSmall   = errors.New("a part other than the last is smaller than the minimum allowed size")
	ErrEntityTooLarge   = errors.New("the part is larger than the maximum allowed size")
	ErrMalformedParts   = errors.New("the part list is empty or a part number is out of range")
)

var uploadIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// MultipartUpload is an upload in progress (upload.json).
type MultipartUpload struct {
	ID        string    `json:"id"`
	Bucket    string    `json:"bucket"`
	BucketID  int       `json:"bucket_id"`
	Key       string    `json:"key"`
	OwnerID   int       `json:"owner_id"`
	Public    bool      `json:"public"`
	Initiated time.Time `json:"initiated"`
}

// CompletedPart is one entry of a CompleteMultipartUpload request.
type CompletedPart struct {
	PartNumber int
	ETag       string // quoted or not
}

func uploadDir(id string) string { return filepath.Join(MultipartRoot, id) }

func partPath(id string, n int) string {
	return filepath.Join(uploadDir(id), fmt.Sprintf("part-%05d", n))
}

// CreateMultipartUpload starts an upload for key and returns it.
func CreateMultipartUpload(bucket types.Bucket, key string, user *types.Authorization, public bool) (*MultipartUpload, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	up := &MultipartUpload{
		ID:        hex.EncodeToString(b),
		Bucket:    bucket.Name,
		BucketID:  bucket.ID,
		Key:       key,
		OwnerID:   user.ID,
		Public:    public,
		Initiated: time.Now().UTC(),
	}
	if err := os.MkdirAll(uploadDir(up.ID), 0o755); err != nil {
		return nil, err
	}
	meta, err := json.Marshal(up)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(filepath.Join(uploadDir(up.ID), "upload.json"), meta); err != nil {
		os.RemoveAll(uploadDir(up.ID))
		return nil, err
	}
	return up, nil
}

// LoadMultipartUpload returns the upload with this id, which must belong to
// bucket/key (an upload id names exactly one object, as on S3).
func LoadMultipartUpload(id, bucket, key string) (*MultipartUpload, error) {
	if !uploadIDPattern.MatchString(id) {
		return nil, ErrNoSuchUpload
	}
	raw, err := os.ReadFile(filepath.Join(uploadDir(id), "upload.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoSuchUpload
	}
	if err != nil {
		return nil, err
	}
	var up MultipartUpload
	if err := json.Unmarshal(raw, &up); err != nil {
		return nil, fmt.Errorf("read upload %s: %w", id, err)
	}
	if up.Bucket != bucket || up.Key != key {
		return nil, ErrNoSuchUpload
	}
	return &up, nil
}

// WriteMultipartPart stores part n of the upload and returns its ETag (the
// part's MD5). Re-sending a part number replaces it, as on S3. A non-nil
// wantMD5 is checked like Content-MD5 on a PUT (ErrBadDigest).
func WriteMultipartPart(up *MultipartUpload, n int, body io.Reader, wantMD5 []byte) (types.ETag, error) {
	if n < 1 || n > MaxPartNumber {
		return "", ErrMalformedParts
	}
	tmp, err := os.CreateTemp(uploadDir(up.ID), ".part-*")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoSuchUpload // aborted meanwhile
		}
		return "", err
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // no-op once renamed
	}()

	w := tools.NewETagWriter(tmp)
	written, err := io.Copy(w, io.LimitReader(body, MaxPartSize+1))
	if err != nil {
		return "", err
	}
	if written > MaxPartSize {
		return "", ErrEntityTooLarge
	}
	if wantMD5 != nil && string(wantMD5) != string(w.Sum()) {
		return "", ErrBadDigest
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	etag := w.ETag()
	// The MD5 sidecar first: a part file is only ever visible with its MD5.
	if err := writeFileAtomic(partPath(up.ID, n)+".md5", []byte(etag)); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, partPath(up.ID, n)); err != nil {
		return "", err
	}
	return etag, nil
}

// MultipartCompletion is a validated CompleteMultipartUpload: every listed
// part exists, matches its ETag and is big enough. Commit joins the parts;
// Close releases the part files either way.
type MultipartCompletion struct {
	up    *MultipartUpload
	files []*os.File
	etag  types.ETag
}

// PrepareMultipartComplete validates a part list. It is quick (no data is
// read), so its errors can still be sent with a proper status code before the
// slow join starts.
func PrepareMultipartComplete(up *MultipartUpload, parts []CompletedPart) (*MultipartCompletion, error) {
	if len(parts) == 0 || len(parts) > MaxPartNumber {
		return nil, ErrMalformedParts
	}
	// Validate the list, then every part's existence and ETag, then sizes, so
	// the most basic problem is the one reported.
	for i, p := range parts {
		if p.PartNumber < 1 || p.PartNumber > MaxPartNumber {
			return nil, ErrMalformedParts
		}
		if i > 0 && p.PartNumber <= parts[i-1].PartNumber {
			return nil, ErrInvalidPartOrder
		}
	}

	c := &MultipartCompletion{up: up}
	ok := false
	defer func() {
		if !ok {
			c.Close()
		}
	}()
	// S3 defines the multipart ETag over the part MD5s; tools.ETagWriter is
	// the one MD5 implementation (ETags are integrity tags, not security).
	sums := tools.NewETagWriter(io.Discard)
	sizes := make([]int64, len(parts))
	for i, p := range parts {
		stored, err := os.ReadFile(partPath(up.ID, p.PartNumber) + ".md5")
		if err != nil || strings.Trim(p.ETag, `"`) != string(stored) {
			return nil, ErrInvalidPart
		}
		f, err := os.Open(partPath(up.ID, p.PartNumber))
		if err != nil {
			return nil, ErrInvalidPart
		}
		c.files = append(c.files, f)
		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		sizes[i] = st.Size()
		raw, err := hex.DecodeString(string(stored))
		if err != nil {
			return nil, ErrInvalidPart
		}
		_, _ = sums.Write(raw) // io.Discard never fails
	}
	for i := 0; i < len(sizes)-1; i++ {
		if sizes[i] < MinPartSize {
			return nil, ErrEntityTooSmall
		}
	}
	c.etag = types.ETag(fmt.Sprintf("%s-%d", sums.ETag(), len(parts)))
	ok = true
	return c, nil
}

// Commit joins the parts into the object at filePath and removes the upload.
// The object's ETag is S3's multipart form, "<hex MD5 of the concatenated
// binary part MD5s>-<part count>".
func (c *MultipartCompletion) Commit(filePath string, bucket types.Bucket, user *types.Authorization) (*types.ETag, error) {
	defer c.Close()
	readers := make([]io.Reader, len(c.files))
	for i, f := range c.files {
		readers[i] = f
	}
	public := c.up.Public
	out, err := createObject(filePath, c.up.Key, bucket, io.MultiReader(readers...), nil, user, &public, nil, &c.etag)
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(uploadDir(c.up.ID)); err != nil {
		log.Println("multipart: object stored but upload dir not removed:", c.up.ID, err)
	}
	return out, nil
}

// Close releases the part files.
func (c *MultipartCompletion) Close() {
	for _, f := range c.files {
		f.Close()
	}
	c.files = nil
}

// CompleteMultipartUpload is PrepareMultipartComplete then Commit.
func CompleteMultipartUpload(up *MultipartUpload, parts []CompletedPart, filePath string, bucket types.Bucket, user *types.Authorization) (*types.ETag, error) {
	c, err := PrepareMultipartComplete(up, parts)
	if err != nil {
		return nil, err
	}
	return c.Commit(filePath, bucket, user)
}

// AbortMultipartUpload discards the upload and its parts.
func AbortMultipartUpload(up *MultipartUpload) error {
	return os.RemoveAll(uploadDir(up.ID))
}

// CleanupStaleMultipartUploads removes uploads initiated more than maxAge ago
// and returns how many it removed.
func CleanupStaleMultipartUploads(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(MultipartRoot)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if !e.IsDir() || !uploadIDPattern.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		started := info.ModTime()
		if raw, err := os.ReadFile(filepath.Join(uploadDir(e.Name()), "upload.json")); err == nil {
			var up MultipartUpload
			if json.Unmarshal(raw, &up) == nil && !up.Initiated.IsZero() {
				started = up.Initiated
			}
		}
		if started.Before(cutoff) {
			if err := os.RemoveAll(uploadDir(e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
