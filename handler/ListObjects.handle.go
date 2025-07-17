package handler

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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

	all, err := ListObjects(bucket)
	if err != nil {
		return nil, err
	}

	var (
		contents []types.ObjectMetadata
		cpMap    = make(map[string]struct{})
	)

	for _, obj := range all {
		if !strings.HasPrefix(obj.Key, prefix) {
			continue
		}

		if delimiter == "/" {
			trim := strings.TrimPrefix(obj.Key, prefix)
			trim = strings.TrimPrefix(trim, "/")

			if trim == "" {

				continue
			}

			if i := strings.IndexByte(trim, '/'); i != -1 {
				cpMap[prefix+trim[:i+1]] = struct{}{}
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

	return &types.ObjectList{
		Name:           bucket,
		Prefix:         prefix,
		Delimiter:      delimiter,
		MaxKeys:        1000,
		IsTruncated:    false,
		Contents:       contents,
		CommonPrefixes: cps,
	}, nil
}

func isIgnored(name string) bool {
	if name == ".DS_Store" {
		return true
	}
	return false
}
