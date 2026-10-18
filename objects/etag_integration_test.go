//go:build integration

package objects

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// etagTestSetup seeds the walk bucket and runs the test from a temp dir, so
// buckets/<bucket>/... lands there.
func etagTestSetup(t *testing.T) (types.Bucket, *types.Authorization) {
	t.Helper()
	bucketID := seedWalkBucket(t)
	var ownerID int
	db.DB.QueryRow(`SELECT owner_id FROM buckets WHERE id = $1`, bucketID).Scan(&ownerID)
	wd, _ := os.Getwd()
	os.Chdir(t.TempDir())
	t.Cleanup(func() { os.Chdir(wd) })
	return types.Bucket{ID: bucketID, Name: walkTestBucket}, &types.Authorization{ID: ownerID}
}

func put(t *testing.T, b types.Bucket, user *types.Authorization, key, body string) types.ETag {
	t.Helper()
	etag, err := CreateObject(filepath.Join("buckets", b.Name, key), key, b, strings.NewReader(body), nil, user, nil)
	if err != nil {
		t.Fatal(err)
	}
	return *etag
}

// The stored, returned, fetched and listed ETag is the MD5 of the body, and
// identical bytes under two keys share it.
func TestETagIsContentMD5(t *testing.T) {
	b, user := etagTestSetup(t)
	cases := []struct{ key, body string }{
		{"etag/hello.txt", "hello"},
		{"etag/copy-of-hello.txt", "hello"},
		{"etag/deep/other.bin", "something else entirely"},
		{"etag/empty", ""},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			want := md5hex(c.body)
			if got := put(t, b, user, c.key, c.body); string(got) != want {
				t.Fatalf("CreateObject ETag = %s, want %s", got, want)
			}
			obj, err := GetObject(b.Name, c.key, nil)
			if err != nil || obj == nil || obj.ETag == nil {
				t.Fatalf("GetObject: %v %v", obj, err)
			}
			if string(*obj.ETag) != want {
				t.Fatalf("stored ETag = %s, want %s", *obj.ETag, want)
			}
			if obj.ETag.Quoted() != `"`+want+`"` {
				t.Fatalf("header ETag = %s", obj.ETag.Quoted())
			}
		})
	}
	if md5hex("hello") != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatal("md5 sanity")
	}

	listed, err := ListObjects(b.Name, &ListObjectsOptions{BucketID: b.ID})
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, o := range listed {
		for _, c := range cases {
			if o.Key == c.key {
				seen++
				if string(*o.ETag) != md5hex(c.body) {
					t.Fatalf("listed ETag for %s = %s, want %s", o.Key, *o.ETag, md5hex(c.body))
				}
			}
		}
	}
	if seen != len(cases) {
		t.Fatalf("listing returned %d of %d keys", seen, len(cases))
	}

	if err := os.MkdirAll(filepath.Join("buckets", b.Name, "etag/folder/"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CreateFolder(filepath.Join("buckets", b.Name, "etag/folder/"), "etag/folder/", b, *user); err != nil {
		t.Fatal(err)
	}
	folder, _ := GetObject(b.Name, "etag/folder/", nil)
	if folder == nil || string(*folder.ETag) != md5hex("") {
		t.Fatalf("folder marker ETag = %v, want md5 of empty", folder)
	}
}

// A body that doesn't match Content-MD5 is refused and the stored object is
// left as it was, bytes and row.
func TestCreateObjectBadDigestKeepsExisting(t *testing.T) {
	b, user := etagTestSetup(t)
	put(t, b, user, "digest/a.txt", "original")

	wrong := md5.Sum([]byte("not the body"))
	_, err := CreateObjectVerified(filepath.Join("buckets", b.Name, "digest/a.txt"), "digest/a.txt", b, strings.NewReader("replacement"), nil, user, nil, wrong[:])
	if !errors.Is(err, ErrBadDigest) {
		t.Fatalf("err = %v, want ErrBadDigest", err)
	}
	data, _ := os.ReadFile(filepath.Join("buckets", b.Name, "digest/a.txt"))
	if string(data) != "original" {
		t.Fatalf("file now %q", data)
	}
	obj, _ := GetObject(b.Name, "digest/a.txt", nil)
	if string(*obj.ETag) != md5hex("original") {
		t.Fatalf("row ETag changed to %s", *obj.ETag)
	}
	entries, _ := os.ReadDir(filepath.Join("buckets", b.Name, "digest"))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}

	right := md5.Sum([]byte("replacement"))
	if _, err := CreateObjectVerified(filepath.Join("buckets", b.Name, "digest/a.txt"), "digest/a.txt", b, strings.NewReader("replacement"), nil, user, nil, right[:]); err != nil {
		t.Fatalf("matching digest refused: %v", err)
	}
}

// Old-scheme ETags are rewritten to the content MD5; dry runs write nothing.
func TestBackfillETags(t *testing.T) {
	b, user := etagTestSetup(t)
	db.DB.Exec(`DELETE FROM objects WHERE bucket_id = $1`, b.ID)
	db.DB.Exec(`INSERT INTO objects(bucket_id, key, etag, owner_id, last_modified, uploaded_at) VALUES ($1, 'fill/gone.txt', 'oldscheme', $2, now(), now())`, b.ID, user.ID)
	put(t, b, user, "fill/a.txt", "alpha")
	put(t, b, user, "fill/b.txt", "alpha")
	db.DB.Exec(`UPDATE objects SET etag = 'oldscheme' WHERE bucket_id = $1 AND key IN ('fill/a.txt', 'fill/b.txt')`, b.ID)

	res, err := BackfillETags(b.Name, true)
	if err != nil {
		t.Fatal(err)
	}
	// A row with no file is reported, not touched.
	if res.Updated != 2 || len(res.Missing) != 1 || res.Missing[0] != b.Name+"/fill/gone.txt" {
		t.Fatalf("dry run: %+v", res)
	}
	obj, _ := GetObject(b.Name, "fill/a.txt", nil)
	if *obj.ETag != "oldscheme" {
		t.Fatal("dry run wrote")
	}

	if res, err = BackfillETags(b.Name, false); err != nil || res.Updated != 2 {
		t.Fatalf("backfill: %+v %v", res, err)
	}
	for _, k := range []string{"fill/a.txt", "fill/b.txt"} {
		obj, _ := GetObject(b.Name, k, nil)
		if string(*obj.ETag) != md5hex("alpha") {
			t.Fatalf("%s ETag = %s", k, *obj.ETag)
		}
	}
	if res, _ = BackfillETags(b.Name, false); res.Updated != 0 || res.Unchanged != 2 {
		t.Fatalf("second run not idempotent: %+v", res)
	}
}
