package objects

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/tools"
)

// BackfillETagsResult counts what BackfillETags did.
type BackfillETagsResult struct {
	Scanned   int      // rows read
	Updated   int      // rows whose ETag changed (or would change, in a dry run)
	Unchanged int      // rows already holding the content MD5
	Missing   []string // "bucket/key" rows with no file on disk; left as they are
}

// BackfillETags recomputes every object's ETag from its file on disk
// (buckets/<bucket>/<key>, relative to the working directory) as the content
// MD5. ETags written before 2026-10-18 were md5(path + content). Rows are
// read a batch at a time and the batch's rows closed before any update. Only
// the etag column changes — last_modified is left alone. bucketName "" means
// every bucket; dryRun reports without writing.
func BackfillETags(bucketName string, dryRun bool) (*BackfillETagsResult, error) {
	const batch = 500
	res := &BackfillETagsResult{}
	lastID := 0
	for {
		q := db.Psql.Select("objects.id", "buckets.name", "objects.key", "objects.etag").
			From("objects").Join("buckets ON buckets.id = objects.bucket_id").
			Where(sq.Gt{"objects.id": lastID}).OrderBy("objects.id").Limit(batch)
		if bucketName != "" {
			q = q.Where(sq.Eq{"buckets.name": bucketName})
		}
		rows, err := q.Query()
		if err != nil {
			return res, err
		}
		type row struct {
			id               int
			bucket, key, tag string
		}
		var page []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.bucket, &r.key, &r.tag); err != nil {
				rows.Close()
				return res, err
			}
			page = append(page, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return res, err
		}
		if len(page) == 0 {
			return res, nil
		}

		for _, r := range page {
			lastID = r.id
			res.Scanned++
			var etag string
			if strings.HasSuffix(r.key, "/") {
				etag = string(tools.ETagOf(nil)) // folder marker: zero bytes
			} else {
				e, err := tools.GenerateETag(filepath.Join("buckets", r.bucket, r.key))
				if errors.Is(err, os.ErrNotExist) {
					res.Missing = append(res.Missing, r.bucket+"/"+r.key)
					continue
				}
				if err != nil {
					return res, err
				}
				etag = string(e)
			}
			if etag == r.tag {
				res.Unchanged++
				continue
			}
			res.Updated++
			if dryRun {
				continue
			}
			if _, err := db.Psql.Update("objects").Set("etag", etag).Where(sq.Eq{"id": r.id}).Exec(); err != nil {
				return res, err
			}
		}
	}
}
