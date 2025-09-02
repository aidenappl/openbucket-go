package handler

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/types"
)

func ListObjectsXML(bucket string, q url.Values) (*types.ObjectList, error) {

	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	encodingType := q.Get("encoding-type")
	maxKeys := 1000

	if v := q.Get("max-keys"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxKeys = n
		}
	}

	all, err := objects.ListObjects(bucket)
	if err != nil {
		return nil, err
	}

	normPrefix := strings.TrimPrefix(prefix, "/")

	var contents []types.ObjectMetadata
	cpMap := make(map[string]struct{})

	for _, obj := range all {
		key := obj.Key

		// Filter by prefix
		if normPrefix != "" && !strings.HasPrefix(key, normPrefix) {
			continue
		}

		if delimiter != "" {
			// Remainder after prefix
			trim := strings.TrimPrefix(key, normPrefix)
			trim = strings.TrimPrefix(trim, "/") // if user supplied a trailing '/'

			// If key equals prefix (or is the placeholder of it), skip contents entry
			if trim == "" {
				// This covers exact match with a "directory key" like "aiden/".
				// Contribute a common prefix if appropriate
				if delimiter == "/" && strings.HasSuffix(key, "/") {
					// For a placeholder exactly equal to prefix, it represents the prefix itself; do not emit CP
				}
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
		} else {
			// No delimiter → return everything (including placeholder keys)
		}

		contents = append(contents, obj)
	}

	// Sort results
	sort.Slice(contents, func(i, j int) bool { return contents[i].Key < contents[j].Key })

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
			// ETag should stay quoted string; don’t encode it
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
