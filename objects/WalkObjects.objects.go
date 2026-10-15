package objects

import (
	"fmt"
	"unicode/utf8"

	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/listing"
)

// walkQuery returns, in byte order, one key per listing entry under a prefix:
// each leaf key, and the FIRST key of each common prefix. It walks the
// (bucket_id, key COLLATE "C") index with one probe per entry ("loose index
// scan"): after a leaf it takes the next key; after a key inside a common
// prefix G it jumps to the first key >= Successor(G), skipping G's contents.
//
// A page of a folder with 3,000 sub-folders holding 38,000 objects is ~1,000
// index probes instead of reading all 38,000 rows (2.2 s in production).
//
// Parameters:
//
//	$1 bucket id
//	$2 exclusive lower bound (a key), '' for none
//	$3 inclusive lower bound, '' for none
//	$4 prefix
//	$5 Successor(prefix), NULL for no upper bound
//	$6 prefix length in characters (Postgres string functions count characters)
//	$7 delimiter
//	$8 Successor(delimiter): G ends with the delimiter, so Successor(G) is G
//	   with the delimiter replaced by this
//	$9 how many keys to return
//
// The outer LIMIT also stops the recursion: Postgres evaluates a WITH query
// only as far as its parent fetches.
const walkQuery = `
WITH RECURSIVE walk(key) AS (
	(SELECT o.key COLLATE "C"
	   FROM objects o
	  WHERE o.bucket_id = $1
	    AND o.key COLLATE "C" > $2::text
	    AND o.key COLLATE "C" >= $3::text
	    AND o.key COLLATE "C" >= $4::text
	    AND ($5::text IS NULL OR o.key COLLATE "C" < $5)
	  ORDER BY o.key COLLATE "C"
	  LIMIT 1)
	UNION ALL
	SELECT (SELECT o.key COLLATE "C"
	          FROM objects o
	         WHERE o.bucket_id = $1
	           AND o.key COLLATE "C" > w.key
	           AND o.key COLLATE "C" >= CASE
	                 WHEN strpos(substr(w.key, $6::int + 1), $7::text) > 0
	                 THEN left(w.key, $6::int + strpos(substr(w.key, $6::int + 1), $7::text) - 1) || $8::text
	                 ELSE w.key
	               END
	           AND ($5::text IS NULL OR o.key COLLATE "C" < $5)
	         ORDER BY o.key COLLATE "C"
	         LIMIT 1) COLLATE "C"
	  FROM walk w
	 WHERE w.key IS NOT NULL
)
SELECT key FROM walk WHERE key IS NOT NULL LIMIT $9::int`

// walkDelimited returns up to limit keys from walkQuery: leaves after `after`,
// and one key per common prefix that sorts after `after`. A common prefix
// that `after` is inside of (or equal to) was finished by an earlier page and
// is skipped whole.
func walkDelimited(bucketID int, prefix, delimiter, after string, limit int) ([]string, error) {
	delimEnd, ok := listing.Successor(delimiter)
	if !ok {
		return nil, fmt.Errorf("delimiter %q has no successor", delimiter)
	}

	from := ""
	if g, inGroup := listing.GroupOf(after, prefix, delimiter); inGroup {
		if end, ok := listing.Successor(g); ok {
			from = end
		}
	}

	var prefixEnd *string
	if end, ok := listing.Successor(prefix); ok {
		prefixEnd = &end
	}

	rows, err := db.Timed.Query(walkQuery,
		bucketID, after, from, prefix, prefixEnd,
		utf8.RuneCountInString(prefix), delimiter, delimEnd, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to walk objects: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("failed to scan walked key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
