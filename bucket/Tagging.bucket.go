package bucket

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/types"
)

func taggingPath(bucket string) string {
	return filepath.Join("buckets", bucket, ".obtagging")
}

func LoadBucketTags(bucket string) ([]types.Tag, bool, error) {
	p := taggingPath(bucket)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil // no tags
		}
		return nil, false, err
	}
	var t types.BucketTagging
	if err := xml.Unmarshal(b, &t); err != nil {
		return nil, false, err
	}
	return t.TagSet, true, nil
}

func SaveBucketTags(bucket string, tags []types.Tag) error {
	// enforce S3-ish rules
	seen := map[string]struct{}{}
	out := make([]types.Tag, 0, len(tags))
	for _, tg := range tags {
		k := strings.TrimSpace(tg.Key)
		if k == "" {
			continue
		} // skip empty keys
		if _, dup := seen[k]; dup {
			continue
		} // de-dupe by key
		seen[k] = struct{}{}
		out = append(out, types.Tag{Key: k, Value: tg.Value})
	}

	bt := types.BucketTagging{
		Xmlns:  "http://s3.amazonaws.com/doc/2006-03-01/",
		TagSet: out,
	}
	// atomic write
	tmp := taggingPath(bucket) + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return err
	}
	buf, err := xml.MarshalIndent(bt, "", "  ")
	if err != nil {
		return err
	}
	buf = append([]byte(xml.Header), buf...)
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, taggingPath(bucket))
}

func DeleteBucketTags(bucket string) error {
	if err := os.Remove(taggingPath(bucket)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
