package objects

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/aidenappl/openbucket-go/types"
)

func ListObjectsXML(bucket string, q url.Values) (*types.ObjectList, error) {

	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	encodingType := q.Get("encoding-type")
	startAfter := q.Get("start-after")
	if startAfter == "" {
		startAfter = q.Get("marker")
	}
	maxKeys := 1000

	if v := q.Get("max-keys"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxKeys = n
		}
	}

	normPrefix := strings.TrimPrefix(prefix, "/")

	opts := &ListObjectsOptions{
		Prefix:     normPrefix,
		StartAfter: startAfter,
	}

	// When no delimiter, SQL can handle the full limit
	if delimiter == "" {
		opts.Limit = maxKeys
	}

	all, err := ListObjects(bucket, opts)
	if err != nil {
		return nil, err
	}

	var contents []types.ObjectMetadata
	cpMap := make(map[string]struct{})

	for _, obj := range all {
		key := obj.Key

		// Check against isIgnored
		if isIgnored(key) {
			continue
		}

		if delimiter != "" {
			// Remainder after prefix
			trim := strings.TrimPrefix(key, normPrefix)
			trim = strings.TrimPrefix(trim, "/") // if user supplied a trailing '/'

			// If key equals prefix (or is the placeholder of it), skip contents entry
			if trim == "" {
				continue
			}

			// If there's another delimiter in the remainder, bucket it as a CommonPrefix
			if i := strings.Index(trim, delimiter); i != -1 {
				cp := normPrefix
				if cp != "" && !strings.HasSuffix(cp, "/") && delimiter == "/" {
					cp += "/"
				}
				cp += trim[:i+len(delimiter)]
				// Ensure single slash normalization
				cp = strings.ReplaceAll(cp, "//", "/")
				cpMap[cp] = struct{}{}
				continue
			}

			// No further delimiter in remainder → it's a leaf under this level → keep in contents
			// Also exclude directory placeholders from contents when delimiter is active
			if strings.HasSuffix(key, "/") {
				continue
			}
		}

		contents = append(contents, obj)
	}

	var cps []types.CommonPrefix
	for p := range cpMap {
		cps = append(cps, types.CommonPrefix{Prefix: p})
	}
	sort.Slice(cps, func(i, j int) bool { return cps[i].Prefix < cps[j].Prefix })

	// Truncate to maxKeys (v1 uses MaxKeys + IsTruncated + NextMarker when needed)
	isTruncated := false
	if len(contents)+len(cps) > maxKeys {
		isTruncated = true
		// Prefer returning contents first up to budget; then CPs
		budget := maxKeys
		if budget < len(contents) {
			contents = contents[:budget]
			cps = nil
		} else {
			budget -= len(contents)
			if budget < len(cps) {
				cps = cps[:budget]
			}
		}
	}

	// Handle encoding-type=url
	if strings.EqualFold(encodingType, "url") {
		for i := range contents {
			contents[i].Key = url.PathEscape(contents[i].Key)
			// ETag should stay quoted string; do not encode it
		}
		for i := range cps {
			// S3 encodes CommonPrefixes using the same URL encoding
			cps[i].Prefix = url.PathEscape(cps[i].Prefix)
		}
	}

	// Build response
	resp := &types.ObjectList{
		Name:           bucket,
		Prefix:         prefix,
		Delimiter:      delimiter,
		MaxKeys:        maxKeys,
		KeyCount:       len(contents),
		IsTruncated:    isTruncated,
		Contents:       contents,
		CommonPrefixes: cps,
	}

	// If encoding requested, set element
	if strings.EqualFold(encodingType, "url") {
		resp.EncodingType = "url" // add this field in your struct with `xml:"EncodingType,omitempty"`
	}

	return resp, nil
}

func isIgnored(name string) bool {
	if name == ".DS_Store" {
		return true
	}
	return false
}
