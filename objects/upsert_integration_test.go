//go:build integration

package objects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// Existing duplicates are collapsed to the newest row (its tags kept, the
// others' dropped) and the unique index is created.
func TestMigrationCollapsesDuplicateKeys(t *testing.T) {
	bucketID := seedWalkBucket(t)

	if _, err := db.DB.Exec(`DROP INDEX IF EXISTS uniq_objects_bucket_key`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.RunMigrations() })

	var ownerID int
	db.DB.QueryRow(`SELECT owner_id FROM buckets WHERE id = $1`, bucketID).Scan(&ownerID)
	ids := map[string]int{}
	for _, v := range []struct{ name, modified string }{
		{"old", "2025-12-22 18:34:16"}, {"newest", "2025-12-24 05:04:34"}, {"mid", "2025-12-24 04:56:07"},
	} {
		var id int
		if err := db.DB.QueryRow(`INSERT INTO objects(bucket_id, key, etag, owner_id, last_modified, uploaded_at)
			VALUES ($1, 'dup/IMG.jpg', $2, $3, $4, $4) RETURNING id`, bucketID, v.name, ownerID, v.modified).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[v.name] = id
		if _, err := db.DB.Exec(`INSERT INTO object_tags(bucket_id, object_id, tag_key, tag_value) VALUES ($1, $2, 'from', $3)`, bucketID, id, v.name); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.RunMigrations(); err != nil {
		t.Fatal(err)
	}

	var etag string
	var n int
	db.DB.QueryRow(`SELECT count(*), max(etag) FROM objects WHERE bucket_id = $1 AND key = 'dup/IMG.jpg'`, bucketID).Scan(&n, &etag)
	if n != 1 || etag != "newest" {
		t.Fatalf("got %d rows (etag %q), want 1 row, the newest", n, etag)
	}
	var tags []string
	rows, _ := db.DB.Query(`SELECT tag_value FROM object_tags WHERE bucket_id = $1 ORDER BY 1`, bucketID)
	for rows.Next() {
		var v string
		rows.Scan(&v)
		tags = append(tags, v)
	}
	rows.Close()
	if strings.Join(tags, ",") != "newest" {
		t.Fatalf("tags left: %v, want only the kept row's", tags)
	}
	// The removed rows are kept for restore.
	db.DB.QueryRow(`SELECT count(*) FROM objects_superseded WHERE id IN ($1, $2)`, ids["old"], ids["mid"]).Scan(&n)
	if n != 2 {
		t.Fatalf("objects_superseded holds %d of the 2 removed rows", n)
	}
	db.DB.QueryRow(`SELECT count(*) FROM object_tags_superseded WHERE object_id IN ($1, $2)`, ids["old"], ids["mid"]).Scan(&n)
	if n != 2 {
		t.Fatalf("object_tags_superseded holds %d of the 2 removed tags", n)
	}
	// Other keys are untouched.
	db.DB.QueryRow(`SELECT count(*) FROM objects WHERE bucket_id = $1`, bucketID).Scan(&n)
	if want := len(walkTestKeys()) + 1; n != want {
		t.Fatalf("bucket has %d rows, want %d", n, want)
	}
	// And the index now refuses a second row for a key.
	if _, err := db.DB.Exec(`INSERT INTO objects(bucket_id, key, etag, owner_id, last_modified, uploaded_at)
		VALUES ($1, 'dup/IMG.jpg', 'x', $2, now(), now())`, bucketID, ownerID); err == nil {
		t.Fatal("a duplicate key was accepted")
	}
}

// Writing a key twice (an overwrite) leaves one row describing the last write.
func TestCreateObjectOverwriteUpserts(t *testing.T) {
	bucketID := seedWalkBucket(t)
	var ownerID int
	db.DB.QueryRow(`SELECT owner_id FROM buckets WHERE id = $1`, bucketID).Scan(&ownerID)

	wd, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(wd) })

	b := types.Bucket{ID: bucketID, Name: walkTestBucket}
	user := &types.Authorization{ID: ownerID}
	path := filepath.Join("buckets", walkTestBucket, "over", "write.txt")
	pub := true
	for _, body := range []string{"first version", "second, longer version"} {
		if _, err := CreateObject(path, "over/write.txt", b, strings.NewReader(body), nil, user, &pub); err != nil {
			t.Fatal(err)
		}
		pub = false
	}
	if err := CreateFolder(path, "over/write.txt", b, *user); err != nil {
		t.Fatal(err) // marker-style re-create is idempotent too
	}

	var n int
	var size int64
	db.DB.QueryRow(`SELECT count(*), max(size) FROM objects WHERE bucket_id = $1 AND key = 'over/write.txt'`, bucketID).Scan(&n, &size)
	if n != 1 {
		t.Fatalf("got %d rows for one key, want 1", n)
	}
}
