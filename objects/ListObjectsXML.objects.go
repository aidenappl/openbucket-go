package objects

import (
	"net/url"
	"strconv"
	"strings"

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
// With a delimiter many rows roll up into one common prefix, so rows can't be
// bounded up front and everything under the prefix is read. Without one,
// every row is at most one entry: rows are fetched in batches of max-keys+1
// (the +1 shows whether more exist), and another batch is read whenever rows
// were dropped by the exact-prefix or ignore filters and the page came up
// short — otherwise such a page would claim to be the last.
func loadPage(bucket, prefix, delimiter, after string, maxKeys int) (listing.Page, error) {
	if delimiter != "" {
		rows, err := ListObjects(bucket, &ListObjectsOptions{Prefix: prefix, StartAfter: after})
		if err != nil {
			return listing.Page{}, err
		}
		return listing.Paginate(rows, prefix, delimiter, after, maxKeys), nil
	}

	limit := maxKeys + 1
	var rows []types.ObjectMetadata
	cursor := after
	for {
		batch, err := ListObjects(bucket, &ListObjectsOptions{Prefix: prefix, StartAfter: cursor, Limit: limit})
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
