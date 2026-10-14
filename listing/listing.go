// Package listing implements S3 ListObjects paging over key-ordered rows. It
// has no database or env dependencies so it can be unit-tested; objects/
// ListObjectsXML feeds it rows and serialises the page.
package listing

import (
	"encoding/base64"
	"errors"
	"sort"
	"strings"

	"github.com/aidenappl/openbucket-go/types"
)

// ErrInvalidToken is returned for a continuation-token this server didn't
// issue. Handlers map it to 400 InvalidArgument.
var ErrInvalidToken = errors.New("the continuation token provided is incorrect")

// Page is one page of a listing.
type Page struct {
	Contents       []types.ObjectMetadata
	CommonPrefixes []types.CommonPrefix
	Truncated      bool
	// Next is the last entry returned (a key or a common prefix) when
	// Truncated; the next page starts after it.
	Next string
}

// Paginate turns rows (in key order, keys > after) into one page of a
// listing under prefix, rolling keys up into common prefixes at the delimiter.
//
// Entries — leaf keys and common prefixes — are emitted in one key-ordered
// stream, each common prefix once, and the page is cut after maxKeys of them.
// When `after` is a common prefix, every key beneath it was already covered
// by that prefix and is skipped.
func Paginate(rows []types.ObjectMetadata, prefix, delimiter, after string, maxKeys int) Page {
	sorted := rows
	if !sort.SliceIsSorted(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key }) {
		sorted = append([]types.ObjectMetadata(nil), rows...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	}

	var page Page
	emitted := 0
	lastCP := ""

	for _, obj := range sorted {
		key := obj.Key
		// The SQL filter is LIKE prefix%, where `_` and `%` are wildcards:
		// `blog_posts/` also matched `blog-posts/`. Check the real prefix.
		if !strings.HasPrefix(key, prefix) || IsIgnored(key) {
			continue
		}
		if after != "" && key <= after {
			continue
		}

		entry := key
		isPrefix := false
		if delimiter != "" {
			rest := key[len(prefix):]
			if rest == "" {
				continue // the folder placeholder for prefix itself
			}
			if i := strings.Index(rest, delimiter); i != -1 {
				entry = prefix + rest[:i+len(delimiter)]
				isPrefix = true
			}
		}

		if isPrefix {
			if entry == lastCP {
				continue // still inside the prefix just emitted
			}
			// Covered by a previous page that ended on this prefix (or on a
			// key past it).
			if after != "" && entry <= after {
				continue
			}
		}

		if emitted == maxKeys {
			page.Truncated = true
			break
		}

		if isPrefix {
			page.CommonPrefixes = append(page.CommonPrefixes, types.CommonPrefix{Prefix: entry})
			lastCP = entry
		} else {
			page.Contents = append(page.Contents, obj)
		}
		page.Next = entry
		emitted++
	}

	if !page.Truncated {
		page.Next = ""
	}
	return page
}

// The token is the last returned entry, base64url-encoded so it is opaque and
// survives any characters a key may contain.
func EncodeToken(after string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(after))
}

func DecodeToken(token string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(b) == 0 {
		return "", ErrInvalidToken
	}
	return string(b), nil
}

// IsIgnored reports keys that are never listed.
func IsIgnored(name string) bool {
	return name == ".DS_Store"
}
