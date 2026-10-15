package listing

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/aidenappl/openbucket-go/types"
)

// Keys chosen to stress ordering around the "/" delimiter: '-' and '.' sort
// before '/', '0' after it, so prefixes and leaves interleave.
var testKeys = []string{
	".DS_Store",
	"a-b/x.txt",
	"a.txt",
	"a/",
	"a/1.txt",
	"a/2.txt",
	"a/b/",
	"a/b/c.txt",
	"a/b/d/e.txt",
	"a/c.txt",
	"a/z/deep/file.txt",
	"a0.txt",
	"b/1.txt",
	"blog-posts/x.txt",
	"blog_posts/",
	"blog_posts/banner/1.jpg",
	"blog_posts/readme.md",
	"blogXposts/y.txt",
	"c.txt",
}

func rows(keys []string) []types.ObjectMetadata {
	out := make([]types.ObjectMetadata, len(keys))
	for i, k := range keys {
		out[i].Key = k
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// sqlRows mimics the ListObjects query: LIKE prefix% (with `_` as a wildcard)
// and key > after, in key order.
func sqlRows(all []types.ObjectMetadata, prefix, after string) []types.ObjectMetadata {
	var out []types.ObjectMetadata
	for _, o := range all {
		if !likePrefix(o.Key, prefix) || (after != "" && o.Key <= after) {
			continue
		}
		out = append(out, o)
	}
	return out
}

func likePrefix(key, prefix string) bool {
	if len(key) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if prefix[i] != '_' && prefix[i] != key[i] {
			return false
		}
	}
	return true
}

func entries(p Page) []string {
	var out []string
	for _, c := range p.Contents {
		out = append(out, c.Key)
	}
	for _, cp := range p.CommonPrefixes {
		out = append(out, cp.Prefix)
	}
	sort.Strings(out)
	return out
}

func TestPageListingSinglePage(t *testing.T) {
	all := rows(testKeys)
	tests := []struct {
		name, prefix, delim string
		want                []string
	}{
		{"root with delimiter", "", "/", []string{"a-b/", "a.txt", "a/", "a0.txt", "b/", "blog-posts/", "blogXposts/", "blog_posts/", "c.txt"}},
		{"under a/ with delimiter", "a/", "/", []string{"a/", "a/1.txt", "a/2.txt", "a/b/", "a/c.txt", "a/z/"}},
		{"under a/b/ with delimiter", "a/b/", "/", []string{"a/b/", "a/b/c.txt", "a/b/d/"}},
		{"underscore is literal, not a wildcard", "blog_posts/", "/", []string{"blog_posts/", "blog_posts/banner/", "blog_posts/readme.md"}},
		{"an exact key as the prefix lists that key", "a.txt", "/", []string{"a.txt"}},
		{"recursive under a/", "a/", "", []string{"a/", "a/1.txt", "a/2.txt", "a/b/", "a/b/c.txt", "a/b/d/e.txt", "a/c.txt", "a/z/deep/file.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Paginate(sqlRows(all, tt.prefix, ""), tt.prefix, tt.delim, "", 1000)
			got := entries(p)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got  %v\nwant %v", got, tt.want)
			}
			if p.Truncated || p.Next != "" {
				t.Fatalf("single page should not be truncated (next=%q)", p.Next)
			}
		})
	}
}

// Paging through with any page size must return every entry exactly once,
// matching a single big page — the property the old code broke.
func TestPageListingPagesCoverEverythingOnce(t *testing.T) {
	all := rows(testKeys)
	for _, prefix := range []string{"", "a/", "blog_posts/"} {
		for _, delim := range []string{"/", ""} {
			full := entries(Paginate(sqlRows(all, prefix, ""), prefix, delim, "", 1000))
			for size := 1; size <= len(full)+1; size++ {
				var got []string
				after := ""
				for pages := 0; ; pages++ {
					if pages > len(full)+2 {
						t.Fatalf("prefix=%q delim=%q size=%d: paging did not terminate", prefix, delim, size)
					}
					p := Paginate(sqlRows(all, prefix, after), prefix, delim, after, size)
					if n := len(p.Contents) + len(p.CommonPrefixes); n > size {
						t.Fatalf("page has %d entries, max %d", n, size)
					}
					got = append(got, entries(p)...)
					if !p.Truncated {
						break
					}
					// Round-trip the cursor through the wire token.
					tok := EncodeToken(p.Next)
					dec, err := DecodeToken(tok)
					if err != nil || dec != p.Next {
						t.Fatalf("token round-trip: %q -> %q (%v)", p.Next, dec, err)
					}
					after = dec
				}
				sort.Strings(got)
				if !reflect.DeepEqual(got, full) {
					t.Fatalf("prefix=%q delim=%q size=%d:\n got  %v\n want %v", prefix, delim, size, got, full)
				}
			}
		}
	}
}

func TestPageListingTruncationMarksLastEntry(t *testing.T) {
	all := rows(testKeys)
	p := Paginate(sqlRows(all, "", ""), "", "/", "", 3)
	if !p.Truncated {
		t.Fatal("expected truncation")
	}
	// Ordered stream: a-b/, a.txt, a/ ... so the page ends on the prefix a/.
	if p.Next != "a/" {
		t.Fatalf("next = %q, want %q", p.Next, "a/")
	}
	// The following page must not re-list anything under a/.
	q := Paginate(sqlRows(all, "", p.Next), "", "/", p.Next, 1000)
	for _, e := range entries(q) {
		if strings.HasPrefix(e, "a/") {
			t.Fatalf("entry %q under the previous page's prefix was repeated", e)
		}
	}
}

func TestContinuationTokenRejectsGarbage(t *testing.T) {
	for _, tok := range []string{"!!!", "=", ""} {
		if _, err := DecodeToken(tok); err == nil {
			t.Fatalf("token %q should be rejected", tok)
		}
	}
}

func TestSuccessorBoundsThePrefixRange(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"a/", "a0", true},
		{"hotels/250/", "hotels/2500", true},
		{"/", "0", true},
		{"z", "{", true},
		{"é", "ê", true},
		{"a\U0010FFFF", "b", true},
		{"\uD7FF", "\uE000", true},
		{"", "", false},
		{"\U0010FFFF", "", false},
		{"a\xff", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := Successor(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("Successor(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}

	// Every key with the prefix sorts below the bound; the first key past
	// the prefix range does not.
	for _, p := range []string{"a/", "a", "blog_posts/", "a/b/"} {
		end, _ := Successor(p)
		for _, k := range testKeys {
			if strings.HasPrefix(k, p) != (k >= p && k < end) {
				t.Errorf("prefix %q bound %q misclassifies %q", p, end, k)
			}
		}
	}
}

func TestGroupOf(t *testing.T) {
	tests := []struct {
		key, prefix, delim, want string
		ok                       bool
	}{
		{"a/b/c.txt", "a/", "/", "a/b/", true},
		{"a/c.txt", "a/", "/", "", false},
		{"a/b/", "a/", "/", "a/b/", true},
		{"a/b/c.txt", "", "/", "a/", true},
		{"a/b/c.txt", "x/", "/", "", false},
		{"a/b/c.txt", "a/", "", "", false},
	}
	for _, tt := range tests {
		got, ok := GroupOf(tt.key, tt.prefix, tt.delim)
		if got != tt.want || ok != tt.ok {
			t.Errorf("GroupOf(%q, %q, %q) = %q, %v; want %q, %v", tt.key, tt.prefix, tt.delim, got, ok, tt.want, tt.ok)
		}
	}
}
