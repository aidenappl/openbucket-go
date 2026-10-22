//go:build integration

package objects

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aidenappl/openbucket-go/db"
)

// multipartETag is S3's form: MD5 of the concatenated binary part MD5s, "-N".
// Uses md5hex (etag_integration_test.go): computed independently of the code
// under test.
func multipartETag(parts ...[]byte) string {
	var sums []byte
	for _, p := range parts {
		raw, _ := hex.DecodeString(md5hex(string(p)))
		sums = append(sums, raw...)
	}
	return fmt.Sprintf("%s-%d", md5hex(string(sums)), len(parts))
}

func TestMultipartUpload(t *testing.T) {
	b, user := etagTestSetup(t)
	p1 := bytes.Repeat([]byte("a"), MinPartSize)
	p2 := bytes.Repeat([]byte("b"), MinPartSize+123)
	p3 := []byte("tail")

	t.Run("complete joins the parts with S3's ETag", func(t *testing.T) {
		key := "mp/video.mp4"
		up, err := CreateMultipartUpload(b, key, user, true)
		if err != nil {
			t.Fatal(err)
		}
		// Parts arrive out of order and part 2 is re-sent; the last write wins.
		e3, _ := WriteMultipartPart(up, 3, bytes.NewReader(p3), nil)
		e1, _ := WriteMultipartPart(up, 1, bytes.NewReader(p1), nil)
		WriteMultipartPart(up, 2, bytes.NewReader([]byte("stale")), nil)
		e2, err := WriteMultipartPart(up, 2, bytes.NewReader(p2), nil)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := LoadMultipartUpload(up.ID, b.Name, key)
		if err != nil {
			t.Fatal(err)
		}
		etag, err := CompleteMultipartUpload(loaded, []CompletedPart{
			{1, `"` + string(e1) + `"`}, {2, string(e2)}, {3, string(e3)},
		}, filepath.Join("buckets", b.Name, key), b, user)
		if err != nil {
			t.Fatal(err)
		}
		if want := multipartETag(p1, p2, p3); string(*etag) != want {
			t.Fatalf("ETag = %s, want %s", *etag, want)
		}
		got, _ := os.ReadFile(filepath.Join("buckets", b.Name, key))
		if want := append(append(append([]byte{}, p1...), p2...), p3...); !bytes.Equal(got, want) {
			t.Fatalf("object is %d bytes, want %d", len(got), len(want))
		}
		var dbETag string
		var size int64
		var public bool
		db.DB.QueryRow(`SELECT etag, size, public FROM objects WHERE bucket_id = $1 AND key = $2`, b.ID, key).Scan(&dbETag, &size, &public)
		if dbETag != string(*etag) || size != int64(len(got)) || !public {
			t.Fatalf("row = (%s, %d, %v)", dbETag, size, public)
		}
		if _, err := os.Stat(uploadDir(up.ID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("upload dir left behind after complete")
		}
		if _, err := LoadMultipartUpload(up.ID, b.Name, key); !errors.Is(err, ErrNoSuchUpload) {
			t.Fatalf("completed upload still loads: %v", err)
		}
	})

	t.Run("refusals leave the upload intact", func(t *testing.T) {
		key := "mp/refused.mp4"
		up, _ := CreateMultipartUpload(b, key, user, false)
		e1, _ := WriteMultipartPart(up, 1, bytes.NewReader([]byte("small")), nil)
		e2, _ := WriteMultipartPart(up, 2, bytes.NewReader(p3), nil)
		path := filepath.Join("buckets", b.Name, key)
		cases := []struct {
			name  string
			parts []CompletedPart
			want  error
		}{
			{"too small", []CompletedPart{{1, string(e1)}, {2, string(e2)}}, ErrEntityTooSmall},
			{"wrong etag", []CompletedPart{{1, "deadbeef"}}, ErrInvalidPart},
			{"missing part", []CompletedPart{{1, string(e1)}, {7, string(e2)}}, ErrInvalidPart},
			{"out of order", []CompletedPart{{2, string(e2)}, {1, string(e1)}}, ErrInvalidPartOrder},
			{"duplicate", []CompletedPart{{1, string(e1)}, {1, string(e1)}}, ErrInvalidPartOrder},
			{"empty", nil, ErrMalformedParts},
		}
		for _, c := range cases {
			if _, err := CompleteMultipartUpload(up, c.parts, path, b, user); !errors.Is(err, c.want) {
				t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
			}
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("a refused complete wrote the object")
		}
		// A single small part is a valid upload.
		if _, err := CompleteMultipartUpload(up, []CompletedPart{{1, string(e1)}}, path, b, user); err != nil {
			t.Fatalf("single-part complete: %v", err)
		}
	})

	t.Run("bad digest and bounds", func(t *testing.T) {
		up, _ := CreateMultipartUpload(b, "mp/x", user, false)
		wrong, _ := hex.DecodeString(md5hex("other"))
		if _, err := WriteMultipartPart(up, 1, bytes.NewReader(p3), wrong); !errors.Is(err, ErrBadDigest) {
			t.Fatalf("err = %v, want ErrBadDigest", err)
		}
		for _, n := range []int{0, MaxPartNumber + 1} {
			if _, err := WriteMultipartPart(up, n, bytes.NewReader(p3), nil); !errors.Is(err, ErrMalformedParts) {
				t.Fatalf("part %d: err = %v", n, err)
			}
		}
	})

	t.Run("upload id names one object", func(t *testing.T) {
		up, _ := CreateMultipartUpload(b, "mp/a", user, false)
		for _, args := range [][3]string{
			{up.ID, b.Name, "mp/b"},         // other key
			{up.ID, "other-bucket", "mp/a"}, // other bucket
			{"../../etc", b.Name, "mp/a"},   // not an id
			{"0123456789abcdef0123456789abcdef", b.Name, "mp/a"},
		} {
			if _, err := LoadMultipartUpload(args[0], args[1], args[2]); !errors.Is(err, ErrNoSuchUpload) {
				t.Errorf("Load%v = %v, want ErrNoSuchUpload", args, err)
			}
		}
	})

	t.Run("abort and stale cleanup", func(t *testing.T) {
		up, _ := CreateMultipartUpload(b, "mp/abort", user, false)
		WriteMultipartPart(up, 1, bytes.NewReader(p3), nil)
		if err := AbortMultipartUpload(up); err != nil {
			t.Fatal(err)
		}
		if _, err := WriteMultipartPart(up, 2, bytes.NewReader(p3), nil); !errors.Is(err, ErrNoSuchUpload) {
			t.Fatalf("part after abort: %v", err)
		}

		old, _ := CreateMultipartUpload(b, "mp/old", user, false)
		fresh, _ := CreateMultipartUpload(b, "mp/fresh", user, false)
		old.Initiated = time.Now().Add(-8 * 24 * time.Hour)
		meta := fmt.Sprintf(`{"id":%q,"bucket":%q,"key":"mp/old","initiated":%q}`, old.ID, b.Name, old.Initiated.Format(time.RFC3339))
		os.WriteFile(filepath.Join(uploadDir(old.ID), "upload.json"), []byte(meta), 0o644)
		n, err := CleanupStaleMultipartUploads(7 * 24 * time.Hour)
		if err != nil || n < 1 {
			t.Fatalf("cleanup = (%d, %v)", n, err)
		}
		if _, err := os.Stat(uploadDir(old.ID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("stale upload kept")
		}
		if _, err := os.Stat(uploadDir(fresh.ID)); err != nil {
			t.Fatal("fresh upload removed")
		}
	})
}
