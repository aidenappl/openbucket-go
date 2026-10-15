//go:build integration

// Runs against a real Postgres: DATABASE (with search_path=core),
// PRESIGN_SECRET_KEY and OB_ADMIN_TOKEN must be set. Use a throwaway database
// whose default collation is a locale (e.g. en_US.utf8), as production's is —
// that's what makes byte order matter.
//
//	go test -tags integration ./objects -run Walk
package objects

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/listing"
)

const walkTestBucket = "walk-integration-test"

// Keys that stress byte order around delimiters ('-' and '.' sort before
// '/', '0' after), LIKE wildcards, multi-byte runes, empty segments and
// folder placeholders — plus a few hundred sub-folders to page through.
func walkTestKeys() []string {
	keys := []string{
		".DS_Store", "a-b/x.txt", "a.txt", "a/", "a/1.txt", "a/2.txt", "a/b/",
		"a/b/c.txt", "a/b/d/e.txt", "a/c.txt", "a/z/deep/file.txt", "a0.txt",
		"a//double.txt", "a//", "b/1.txt", "blog-posts/x.txt", "blog_posts/",
		"blog_posts/banner/1.jpg", "blog_posts/readme.md", "blogXposts/y.txt",
		"c.txt", "100%/a.txt", "100x/a.txt", "é/x.txt", "é/y/z.txt", "日本/a.txt",
		"x--y--z.txt", "x--y.txt", "x-.txt", "x---.txt", "hotels/", "hotels/readme.txt",
		"hotels/.DS_Store", "empty-folder/",
	}
	for n := 1; n <= 300; n++ {
		for i := 1; i <= 3; i++ {
			keys = append(keys, fmt.Sprintf("hotels/%d/%d.jpg", n, i))
		}
	}
	return keys
}

func TestMain(m *testing.M) {
	if err := db.RunMigrations(); err != nil {
		fmt.Println("migrations:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func seedWalkBucket(t *testing.T) int {
	t.Helper()
	cleanup := func() {
		db.DB.Exec(`DELETE FROM objects WHERE bucket_id IN (SELECT id FROM buckets WHERE name = $1)`, walkTestBucket)
		db.DB.Exec(`DELETE FROM buckets WHERE name = $1`, walkTestBucket)
		db.DB.Exec(`DELETE FROM authorizations WHERE key_id = $1`, walkTestBucket)
	}
	cleanup()
	t.Cleanup(cleanup)

	var ownerID, bucketID int
	if err := db.DB.QueryRow(`INSERT INTO authorizations(name, key_id, secret_key) VALUES ($1, $2, 'x') RETURNING id`, walkTestBucket, walkTestBucket).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.QueryRow(`INSERT INTO buckets(name, owner_id) VALUES ($1, $2) RETURNING id`, walkTestBucket, ownerID).Scan(&bucketID); err != nil {
		t.Fatal(err)
	}
	for _, k := range walkTestKeys() {
		if _, err := db.DB.Exec(`INSERT INTO objects(bucket_id, key, etag, owner_id, last_modified, uploaded_at) VALUES ($1, $2, 'e', $3, now(), now())`, bucketID, k, ownerID); err != nil {
			t.Fatal(err)
		}
	}
	return bucketID
}

type flatPage struct {
	Contents, Prefixes []string
	Truncated          bool
	Next               string
}

func flatten(p listing.Page) flatPage {
	f := flatPage{Truncated: p.Truncated, Next: p.Next}
	for _, c := range p.Contents {
		f.Contents = append(f.Contents, c.Key)
	}
	for _, c := range p.CommonPrefixes {
		f.Prefixes = append(f.Prefixes, c.Prefix)
	}
	return f
}

// Every page from loadPage must equal the page listing.Paginate cuts from
// all rows under the prefix — the read-everything implementation it replaced.
func TestWalkMatchesFullScanEveryPage(t *testing.T) {
	seedWalkBucket(t)

	all, err := ListObjects(walkTestBucket)
	if err != nil {
		t.Fatal(err)
	}

	prefixes := []string{"", "a", "a/", "a//", "a/b/", "blog_posts/", "100%", "é/", "hotels/", "hotels/2", "hotels/250/", "x-", "nope/", "empty-folder/"}
	for _, delimiter := range []string{"/", "--", ""} {
		for _, prefix := range prefixes {
			for _, maxKeys := range []int{1, 2, 3, 7, 50, 1000} {
				name := fmt.Sprintf("delim=%q prefix=%q max=%d", delimiter, prefix, maxKeys)
				t.Run(name, func(t *testing.T) {
					after := ""
					seen := map[string]bool{}
					for pageNo := 0; ; pageNo++ {
						if pageNo > 2000 {
							t.Fatal("paging did not terminate")
						}
						got, err := loadPage(walkTestBucket, prefix, delimiter, after, maxKeys)
						if err != nil {
							t.Fatal(err)
						}
						want := listing.Paginate(all, prefix, delimiter, after, maxKeys)
						if g, w := flatten(got), flatten(want); !reflect.DeepEqual(g, w) {
							t.Fatalf("page %d after %q:\n got %+v\nwant %+v", pageNo, after, g, w)
						}
						for _, e := range append(flatten(got).Contents, flatten(got).Prefixes...) {
							if seen[e] {
								t.Fatalf("entry %q listed twice", e)
							}
							seen[e] = true
						}
						if !got.Truncated {
							break
						}
						after = got.Next
					}
				})
			}
		}
	}
}

// Atlas's shape: 3,000 hotel folders of ~13 images (39,000 objects).
//
//	go test -tags integration ./objects -run '^$' -bench DelimitedPage
func BenchmarkDelimitedPage(b *testing.B) {
	var ownerID, bucketID int
	db.DB.Exec(`DELETE FROM objects WHERE bucket_id IN (SELECT id FROM buckets WHERE name = 'walk-bench')`)
	db.DB.Exec(`DELETE FROM buckets WHERE name = 'walk-bench'`)
	db.DB.Exec(`DELETE FROM authorizations WHERE key_id = 'walk-bench'`)
	if err := db.DB.QueryRow(`INSERT INTO authorizations(name, key_id, secret_key) VALUES ('walk-bench', 'walk-bench', 'x') RETURNING id`).Scan(&ownerID); err != nil {
		b.Fatal(err)
	}
	if err := db.DB.QueryRow(`INSERT INTO buckets(name, owner_id) VALUES ('walk-bench', $1) RETURNING id`, ownerID).Scan(&bucketID); err != nil {
		b.Fatal(err)
	}
	if _, err := db.DB.Exec(`INSERT INTO objects(bucket_id, key, etag, owner_id, last_modified, uploaded_at)
		SELECT $1, 'hotels/' || h || '/' || i || '.jpg', 'e', $2, now(), now()
		FROM generate_series(1, 3000) h, generate_series(1, 13) i`, bucketID, ownerID); err != nil {
		b.Fatal(err)
	}
	db.DB.Exec(`ANALYZE objects`)
	b.Cleanup(func() {
		db.DB.Exec(`DELETE FROM objects WHERE bucket_id = $1`, bucketID)
		db.DB.Exec(`DELETE FROM buckets WHERE id = $1`, bucketID)
		db.DB.Exec(`DELETE FROM authorizations WHERE id = $1`, ownerID)
	})

	for _, after := range []string{"", "hotels/2000/"} {
		b.Run("walk/after="+after, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := loadPage("walk-bench", "hotels/", "/", after, 1000); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("fullscan/after="+after, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				rows, err := ListObjects("walk-bench", &ListObjectsOptions{Prefix: "hotels/", StartAfter: after})
				if err != nil {
					b.Fatal(err)
				}
				listing.Paginate(rows, "hotels/", "/", after, 1000)
			}
		})
	}
}
