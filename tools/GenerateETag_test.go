package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestETagIsContentMD5(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"hello", "hello", "5d41402abc4b2a76b9719d911017c592"},
		{"empty", "", "d41d8cd98f00b204e9800998ecf8427e"},
	}
	dir := t.TempDir()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewETagWriter(&buf)
			w.Write([]byte(c.body))
			if got := string(w.ETag()); got != c.want {
				t.Fatalf("ETagWriter = %s, want %s", got, c.want)
			}
			if got := string(ETagOf([]byte(c.body))); got != c.want {
				t.Fatalf("ETagOf = %s, want %s", got, c.want)
			}
			// The path must not affect the ETag: same bytes, two paths.
			for _, p := range []string{"a/" + c.name, "b/other-" + c.name} {
				path := filepath.Join(dir, p)
				os.MkdirAll(filepath.Dir(path), 0o755)
				os.WriteFile(path, []byte(c.body), 0o644)
				got, err := GenerateETag(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != c.want {
					t.Fatalf("GenerateETag(%s) = %s, want %s", p, got, c.want)
				}
			}
		})
	}
	t.Run("directory", func(t *testing.T) {
		got, err := GenerateETag(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got != "d41d8cd98f00b204e9800998ecf8427e" {
			t.Fatalf("directory ETag = %s, want md5 of empty", got)
		}
	})
}
