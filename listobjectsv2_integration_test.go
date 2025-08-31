//go:build integration

package main

import (
	"encoding/json"
	"encoding/xml"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/types"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	_, pStr, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(pStr)
	return p
}

func waitHealthy(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get(base + "/")
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server not healthy at %s: %v", base, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func ensureAWS(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("aws"); err != nil {
		t.Skip("aws CLI not found; skipping integration test")
	}
}

func writeXMLWithHeader(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		t.Fatalf("createtmp: %v", err)
	}
	enc := xml.NewEncoder(tmp)
	enc.Indent("", "  ")
	// XML declaration at the very top
	if err := enc.EncodeToken(xml.ProcInst{
		Target: "xml",
		Inst:   []byte(`version="1.0" encoding="UTF-8"`),
	}); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		t.Fatalf("header: %v", err)
	}
	if err := enc.Encode(v); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		t.Fatalf("encode: %v", err)
	}
	if err := enc.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		t.Fatalf("flush: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		t.Fatalf("sync: %v", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		t.Fatalf("close: %v", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		t.Fatalf("rename: %v", err)
	}
	_ = os.Chmod(path, 0o644)
}

func seedBucketPermissions(t *testing.T, bucket, ownerID, ownerName string) {
	t.Helper()
	type Perm = types.Permission // if Permission is a named string type

	perms := types.Bucket{
		XMLName:      xml.Name{Local: "Bucket"},
		Xmlns:        "http://s3.amazonaws.com/doc/2006-03-01/",
		XmlnsXsi:     "http://www.w3.org/2001/XMLSchema-instance", // requires XmlnsXsi field on types.Bucket
		Name:         bucket,
		CreationDate: types.IsoTime(time.Now()),
		ACL:          Perm("PRIVATE"),
		Owner: types.UserObject{
			ID:          ownerID,
			DisplayName: ownerName,
		},
		Grants: []types.Grant{
			{
				Grantee: types.Grantee{
					// field tag on Grantee.Type must use the URI form:
					// `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr,omitempty"`
					Type:        "CanonicalUser",
					ID:          ownerID,
					DisplayName: ownerName,
				},
				Permission: Perm("FULL_CONTROL"),
				DateAdded:  types.IsoTime(time.Now()),
			},
		},
	}
	writeXMLWithHeader(t, filepath.Join("buckets", bucket, ".obpermissions"), perms)
}

func seedBucketTags(t *testing.T, bucket string, tags []types.Tag) {
	t.Helper()
	// Minimal bucket Tagging doc: <Tagging><TagSet>...</TagSet></Tagging>
	type tagSet struct {
		XMLName xml.Name    `xml:"TagSet"`
		Tags    []types.Tag `xml:"Tag"`
	}
	type tagging struct {
		XMLName xml.Name `xml:"Tagging"`
		Xmlns   string   `xml:"xmlns,attr,omitempty"`
		Set     tagSet   `xml:"TagSet"`
	}
	doc := tagging{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Set:   tagSet{Tags: tags},
	}
	writeXMLWithHeader(t, filepath.Join("buckets", bucket, ".obtags"), doc)
}

type authzEntry struct {
	Name   string
	KeyID  string
	Secret string
	When   time.Time
}

func seedAuthorizations(t *testing.T, path string, entries []authzEntry) {
	t.Helper()

	type xmlAuthorization struct {
		Name        string `xml:"Name"`
		KeyID       string `xml:"KEY_ID"`
		SecretKey   string `xml:"SECRET_KEY"`
		DateCreated string `xml:"Date_Created"`
	}
	type xmlAuthorizations struct {
		XMLName        xml.Name           `xml:"Authorizations"`
		Authorizations []xmlAuthorization `xml:"Authorization"`
	}

	doc := xmlAuthorizations{}
	for _, e := range entries {
		doc.Authorizations = append(doc.Authorizations, xmlAuthorization{
			Name:        e.Name,
			KeyID:       e.KeyID,
			SecretKey:   e.Secret,
			DateCreated: e.When.Format(time.RFC3339Nano),
		})
	}

	writeXMLWithHeader(t, path, doc)
}

func Test_ListObjectsV2_AWSCLI(t *testing.T) {
	ensureAWS(t)

	// Isolate filesystem
	tmp := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	// Seed a bucket with one object on disk (your server reads from ./buckets/<name>)
	const bucket = "aplb"
	const key = "AidenID.pdf"

	if err := os.MkdirAll(filepath.Join("buckets", bucket), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("hello world")
	if err := os.WriteFile(filepath.Join("buckets", bucket, key), body, 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed auth/authorizations.xml and make CLI use those keys
	const akid = "3XSsCnzYesfhCzhuWgNdzigKWUeuSkZZ"
	const name = "test"
	const secret = "MIB1VQVlkJTHDliUY1gpZ0dxjMiNko45SejcrPTLVJ2ha0d3Ni8IzKCJMkDfCE4S"
	_ = os.MkdirAll("auth", 0o755)
	seedAuthorizations(t, filepath.Join("buckets", "authorizations.xml"), []authzEntry{
		{Name: name, KeyID: akid, Secret: secret, When: time.Now()},
	})

	// NEW: seed sidecar files so middleware/routes that rely on them are happy
	seedBucketPermissions(t, bucket, akid, name)
	seedBucketTags(t, bucket, nil) // or []types.Tag{{Key:"env", Value:"test"}}

	// Start server on a free port
	port := freePort(t)
	env.Port = strconv.Itoa(port)
	env.Region = "us-east-1" // make sure this matches what your signer expects
	base := "http://127.0.0.1:" + env.Port

	go startServer()
	waitHealthy(t, base)

	// Call aws s3api list-objects-v2
	type listResp struct {
		KeyCount    int  `json:"KeyCount"`
		IsTruncated bool `json:"IsTruncated"`
		Contents    []struct {
			Key          string `json:"Key"`
			Size         int64  `json:"Size"`
			StorageClass string `json:"StorageClass"`
			ETag         ETag   `json:"ETag"` // may be empty if you don't emit it; not asserted
		} `json:"Contents"`
	}

	cmd := exec.Command(
		"aws", "s3api", "list-objects-v2",
		"--endpoint-url", base,
		"--bucket", bucket,
		"--encoding-type", "url",
		"--output", "json",
		"--no-cli-pager",
	)
	// Minimal env for CLI; path-style is important for most S3-compatible servers
	cmd.Env = append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+akid,
		"AWS_SECRET_ACCESS_KEY="+secret,
		"AWS_REGION="+env.Region,
		"AWS_PAGER=",
		"AWS_EC2_METADATA_DISABLED=true",
		"AWS_S3_FORCE_PATH_STYLE=true",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aws list-objects-v2 failed: %v\n%s", err, string(out))
	}

	var got listResp
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v\npayload:\n%s", err, string(out))
	}

	// Assertions: 1 key, correct name & size
	if got.KeyCount != 1 || len(got.Contents) != 1 {
		t.Fatalf("unexpected list: %+v", got)
	}
	if got.Contents[0].Key != key {
		t.Fatalf("expected key %q, got %q", key, got.Contents[0].Key)
	}
	if got.Contents[0].Size != int64(len(body)) {
		t.Fatalf("expected size %d, got %d", len(body), got.Contents[0].Size)
	}
}
