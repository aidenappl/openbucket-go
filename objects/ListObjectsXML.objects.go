package objects

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/listing"
	"github.com/aidenappl/openbucket-go/types"
)

// ErrInvalidContinuationToken is listing.ErrInvalidToken, re-exported for the
// handler.
var ErrInvalidContinuationToken = listing.ErrInvalidToken

// S3 never returns more than 1000 keys per page, whatever max-keys asks for.
const maxKeysCap = 1000

// ListObjectsXML serves ListObjects (v1) and ListObjectsV2 for a bucket.
//
// ⚠️ PAGING. This used to ignore continuation-token and never return a
// NextContinuationToken, so every listing ended at the first page (1000
// entries): clients such as the AWS SDK stop paginating when no token comes
// back. openbucket-api's console showed 994 of atlas's 2,993 hotel folders.
// Paging now follows S3: entries (keys and common prefixes) are ordered
// together by key, a page holds at most max-keys of them, and the token marks
// the last entry returned. See listing.Paginate, which is pure and tested.
func ListObjectsXML(bucket string, q url.Values) (*types.ObjectList, error) {
	prefix := strings.TrimPrefix(q.Get("prefix"), "/")
	delimiter := q.Get("delimiter")
	encodingType := q.Get("encoding-type")
	v2 := q.Get("list-type") == "2"

	maxKeys := maxKeysCap
	if v := q.Get("max-keys"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < maxKeysCap {
			maxKeys = n
		}
	}

	// Where this page starts: after the token's entry (v2), else after
	// start-after (v2) or marker (v1). A token wins over start-after.
	var after string
	token := q.Get("continuation-token")
	if v2 && token != "" {
		decoded, err := listing.DecodeToken(token)
		if err != nil {
			return nil, err
		}
		after = decoded
	} else if v2 {
		after = q.Get("start-after")
	} else {
		after = q.Get("marker")
	}

	page, err := loadPage(bucket, prefix, delimiter, after, maxKeys)
	if err != nil {
		return nil, err
	}

	contents := page.Contents
	cps := page.CommonPrefixes
	if strings.EqualFold(encodingType, "url") {
		for i := range contents {
			contents[i].Key = url.PathEscape(contents[i].Key)
		}
		for i := range cps {
			cps[i].Prefix = url.PathEscape(cps[i].Prefix)
		}
	}

	resp := &types.ObjectList{
		Name:           bucket,
		Prefix:         q.Get("prefix"),
		Delimiter:      delimiter,
		MaxKeys:        maxKeys,
		KeyCount:       len(contents) + len(cps),
		IsTruncated:    page.Truncated,
		Contents:       contents,
		CommonPrefixes: cps,
	}
	if strings.EqualFold(encodingType, "url") {
		resp.EncodingType = "url"
	}
	if v2 {
		resp.ContinuationToken = token
		resp.StartAfter = q.Get("start-after")
		if page.Truncated {
			resp.NextContinuationToken = listing.EncodeToken(page.Next)
		}
	} else {
		resp.Marker = q.Get("marker")
		// S3 only returns NextMarker with a delimiter; without one, clients
		// continue from the last key. Returning it always is harmless.
		if page.Truncated {
			resp.NextMarker = page.Next
		}
	}
	return resp, nil
}

// loadPage reads rows after `after` and cuts one page from them.
//
// Both modes read in batches of max-keys+1 entries (the +1 shows whether more
// exist), and read another batch whenever entries were dropped by the ignore
// filter and the page came up short — otherwise such a page would claim to be
// the last. listing.Paginate cuts the page either way, so both modes share its
// tested semantics.
//
//   - With a delimiter, walkDelimited returns one key per entry (a common
//     prefix is represented by its first key) by jumping over each prefix's
//     contents in the index. It used to read every row under the prefix on
//     every page: atlas/hotels/ (38,000 objects) took 2.2 s per request.
//   - Without one, every row is one entry, read in key order.
func loadPage(bucketName, prefix, delimiter, after string, maxKeys int) (listing.Page, error) {
	b, err := bucket.GetBucket(bucketName, &bucket.GetBucketOptions{IncludeGrants: false})
	if err != nil {
		return listing.Page{}, fmt.Errorf("failed to get bucket %q: %w", bucketName, err)
	}
	if b == nil {
		return listing.Page{}, fmt.Errorf("bucket %q not found", bucketName)
	}

	limit := maxKeys + 1
	var rows []types.ObjectMetadata
	cursor := after

	if delimiter != "" {
		for {
			keys, err := walkDelimited(b.ID, prefix, delimiter, cursor, limit)
			if err != nil {
				return listing.Page{}, err
			}
			// ⚠️ Never call ListObjects with no keys: a nil Keys means "no key
			// filter", and the whole bucket came back as duplicates.
			if len(keys) > 0 {
				batch, err := ListObjects(bucketName, &ListObjectsOptions{BucketID: b.ID, Keys: keys})
				if err != nil {
					return listing.Page{}, err
				}
				rows = append(rows, batch...)
			}
			page := listing.Paginate(rows, prefix, delimiter, after, maxKeys)
			if page.Truncated || len(keys) < limit {
				return page, nil
			}
			// Past the last entry walked; walkDelimited skips the whole
			// common prefix when this key is inside one.
			cursor = keys[len(keys)-1]
		}
	}

	for {
		batch, err := ListObjects(bucketName, &ListObjectsOptions{BucketID: b.ID, Prefix: prefix, StartAfter: cursor, Limit: limit})
		if err != nil {
			return listing.Page{}, err
		}
		rows = append(rows, batch...)
		page := listing.Paginate(rows, prefix, "", after, maxKeys)
		if page.Truncated || len(batch) < limit {
			return page, nil
		}
		cursor = batch[len(batch)-1].Key
	}
}
