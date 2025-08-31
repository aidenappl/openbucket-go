package handler

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aidenappl/openbucket-go/types"
)

func ListObjects(bucket string) ([]types.ObjectMetadata, error) {
	root := filepath.Join("buckets", bucket)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("bucket %q not found", bucket)
	}

	var out []types.ObjectMetadata

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if isIgnored(d.Name()) {
			return nil
		}
		if path == root {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		key := filepath.ToSlash(rel)

		// Handle directory keys
		if d.IsDir() {

			st, _ := os.Stat(path)
			out = append(out, types.ObjectMetadata{
				Key:          key + "/",
				LastModified: types.IsoTime(st.ModTime()),
			})
			return nil
		}

		// Skip metadata files
		if strings.HasSuffix(d.Name(), ".obmeta") {
			return nil
		}

		st, _ := os.Stat(path)
		oc := types.ObjectMetadata{
			Key:          key,
			Size:         st.Size(),
			LastModified: types.IsoTime(st.ModTime()),
		}

		metaPath := path + ".obmeta"
		if f, err := os.Open(metaPath); err == nil {
			defer f.Close()
			var m types.ObjectMetadata
			if err := xml.NewDecoder(f).Decode(&m); err == nil {
				oc.ETag = m.ETag
				oc.Owner = m.Owner
			}
		}
		out = append(out, oc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

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

	all, err := ListObjects(bucket)
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
	if strings.HasPrefix(name, ".ob") {
		return true
	}
	return false
}
