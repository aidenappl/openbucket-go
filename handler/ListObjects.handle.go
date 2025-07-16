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

	"github.com/aidenappl/openbucket-go/metadata"
	"github.com/aidenappl/openbucket-go/types"
)

func ListObjects(bucket string) ([]types.ObjectContent, error) {
	root := filepath.Join("buckets", bucket)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("bucket %q not found", bucket)
	}

	var out []types.ObjectContent

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if isIgnored(d.Name()) {
			return nil // skip
		}
		if path == root { // skip bucket root
			return nil
		}

		rel, _ := filepath.Rel(root, path) // e.g.  "folder/file.txt"
		rel = filepath.ToSlash(rel)        // Windows ➜ unix
		key := rel                         // no leading slash for S3
		if d.IsDir() {
			// directory placeholder: Size 0, LastModified = dir ModTime
			st, _ := os.Stat(path)
			out = append(out, types.ObjectContent{
				Key:          key + "/", // "folder/"
				LastModified: types.IsoTime(st.ModTime()),
				Size:         0,
			})
			return nil // keep walking into sub‑dirs
		}

		// ignore loose *.obmeta
		if strings.HasSuffix(d.Name(), ".obmeta") {
			return nil
		}

		// ---------- real object ----------
		st, _ := os.Stat(path)
		oc := types.ObjectContent{
			Key:          key,
			LastModified: types.IsoTime(st.ModTime()),
			Size:         st.Size(),
		}

		// optional metadata
		metaPath := path + ".obmeta"
		if f, err := os.Open(metaPath); err == nil {
			defer f.Close()
			var m metadata.Metadata
			if err := xml.NewDecoder(f).Decode(&m); err == nil {
				oc.ETag = m.ETag
				oc.LastModified = types.IsoTime(m.LastModified)
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

// ListObjectsXML produces the exact XML expected by list‑objects‑v2.
func ListObjectsXML(bucket string, q url.Values) (*types.ObjectList, error) {

	// recognise standard parameters
	prefix := q.Get("prefix")       // may be ""
	delimiter := q.Get("delimiter") // normally "/"
	// (max‑keys, continuation‑token etc. omitted for brevity)

	all, err := ListObjects(bucket)
	if err != nil {
		return nil, err
	}

	var (
		contents []types.ObjectContent
		cpMap    = make(map[string]struct{})
	)

	for _, obj := range all {
		if !strings.HasPrefix(obj.Key, prefix) {
			continue
		}

		if delimiter == "/" {
			trim := strings.TrimPrefix(obj.Key, prefix)
			trim = strings.TrimPrefix(trim, "/")

			// <-- add this guard -----------------------------------------------
			if trim == "" {
				// it's the prefix itself ("howdo/") – do NOT emit anything
				continue
			}
			//--------------------------------------------------------------------

			if i := strings.IndexByte(trim, '/'); i != -1 {
				cpMap[prefix+trim[:i+1]] = struct{}{}
				continue
			}
		}
		contents = append(contents, obj)
	}

	// build sorted CommonPrefixes
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
	if name == ".DS_Store" { // macOS Finder file
		return true
	}
	return false
}
