package aws

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func crc32b64(b []byte) string {
	sum := crc32.ChecksumIEEE(b)
	return base64.StdEncoding.EncodeToString([]byte{byte(sum >> 24), byte(sum >> 16), byte(sum >> 8), byte(sum)})
}

// frame builds an aws-chunked body: chunks of at most size bytes, optional
// chunk signatures, and the given trailer lines.
func frame(data []byte, size int, signed bool, trailers ...string) []byte {
	var b bytes.Buffer
	ext := func() string {
		if signed {
			return ";chunk-signature=" + strings.Repeat("a", 64)
		}
		return ""
	}
	for len(data) > 0 {
		n := min(size, len(data))
		fmt.Fprintf(&b, "%x%s\r\n", n, ext())
		b.Write(data[:n])
		b.WriteString("\r\n")
		data = data[n:]
	}
	fmt.Fprintf(&b, "0%s\r\n", ext())
	for _, t := range trailers {
		b.WriteString(t + "\r\n")
	}
	b.WriteString("\r\n")
	return b.Bytes()
}

func chunkedRequest(body []byte, headers map[string]string) *http.Request {
	r, _ := http.NewRequest(http.MethodPut, "http://x/b/k", bytes.NewReader(body))
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestChunkedReader(t *testing.T) {
	data := bytes.Repeat([]byte("hillview-thumbnail-"), 9000) // ~170 KB, several chunks
	good := crc32b64(data)
	sha := sha256.Sum256(data)

	tests := []struct {
		name    string
		body    []byte
		headers map[string]string
		wantErr error
	}{
		{
			// What Go SDK v2 sent hillview-core-api's uploads as (2026-10-18).
			name: "unsigned payload with crc32 trailer",
			body: frame(data, len(data), false, "x-amz-checksum-crc32:"+good),
			headers: map[string]string{
				"X-Amz-Content-Sha256":         "STREAMING-UNSIGNED-PAYLOAD-TRAILER",
				"Content-Encoding":             "aws-chunked",
				"X-Amz-Trailer":                "x-amz-checksum-crc32",
				"X-Amz-Decoded-Content-Length": strconv.Itoa(len(data)),
			},
		},
		{
			name:    "signed chunks, no trailer",
			body:    frame(data, 64*1024, true),
			headers: map[string]string{"X-Amz-Content-Sha256": "STREAMING-AWS4-HMAC-SHA256-PAYLOAD", "X-Amz-Decoded-Content-Length": strconv.Itoa(len(data))},
		},
		{
			name: "signed chunks with sha256 trailer",
			body: frame(data, 8192, true, "x-amz-checksum-sha256:"+base64.StdEncoding.EncodeToString(sha[:]), "x-amz-trailer-signature:"+strings.Repeat("b", 64)),
			headers: map[string]string{
				"X-Amz-Content-Sha256": "STREAMING-AWS4-HMAC-SHA256-PAYLOAD-TRAILER",
				"X-Amz-Trailer":        "x-amz-checksum-sha256",
			},
		},
		{
			name:    "unknown checksum algorithm is decoded, not verified",
			body:    frame(data, 4096, false, "x-amz-checksum-crc64nvme:AAAAAAAAAAA="),
			headers: map[string]string{"Content-Encoding": "aws-chunked", "X-Amz-Trailer": "x-amz-checksum-crc64nvme"},
		},
		{
			name:    "empty object",
			body:    frame(nil, 1, false, "x-amz-checksum-crc32:"+crc32b64(nil)),
			headers: map[string]string{"Content-Encoding": "aws-chunked", "X-Amz-Trailer": "x-amz-checksum-crc32", "X-Amz-Decoded-Content-Length": "0"},
		},
		{
			name:    "crc32 mismatch",
			body:    frame(data, len(data), false, "x-amz-checksum-crc32:AAAAAA=="),
			headers: map[string]string{"Content-Encoding": "aws-chunked", "X-Amz-Trailer": "x-amz-checksum-crc32"},
			wantErr: ErrChunkedChecksum,
		},
		{
			name:    "declared trailer missing",
			body:    frame(data, len(data), false),
			headers: map[string]string{"Content-Encoding": "aws-chunked", "X-Amz-Trailer": "x-amz-checksum-crc32"},
			wantErr: ErrChunkedChecksum,
		},
		{
			name:    "decoded length mismatch",
			body:    frame(data, len(data), false),
			headers: map[string]string{"Content-Encoding": "aws-chunked", "X-Amz-Decoded-Content-Length": "5"},
			wantErr: ErrChunkedLength,
		},
		{
			name:    "truncated inside a chunk",
			body:    frame(data, len(data), false)[:1000],
			headers: map[string]string{"Content-Encoding": "aws-chunked"},
			wantErr: ErrChunkedFraming,
		},
		{
			name:    "bad chunk size",
			body:    []byte("zz\r\nhello\r\n0\r\n\r\n"),
			headers: map[string]string{"Content-Encoding": "aws-chunked"},
			wantErr: ErrChunkedFraming,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chunkedRequest(tt.body, tt.headers)
			if !IsAWSChunked(r) {
				t.Fatal("IsAWSChunked = false")
			}
			got, err := io.ReadAll(NewChunkedReader(r))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			want := data
			if strings.HasPrefix(tt.name, "empty") {
				want = nil
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("decoded %d bytes, want %d", len(got), len(want))
			}
		})
	}
}

func TestIsAWSChunked(t *testing.T) {
	tests := []struct {
		headers map[string]string
		want    bool
	}{
		{map[string]string{"X-Amz-Content-Sha256": "UNSIGNED-PAYLOAD"}, false},
		{map[string]string{"X-Amz-Content-Sha256": strings.Repeat("a", 64)}, false},
		{map[string]string{"Content-Encoding": "gzip"}, false},
		{map[string]string{"Content-Encoding": "gzip, aws-chunked"}, true},
		{map[string]string{"X-Amz-Content-Sha256": "STREAMING-UNSIGNED-PAYLOAD-TRAILER"}, true},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if got := IsAWSChunked(chunkedRequest(nil, tt.headers)); got != tt.want {
				t.Fatalf("IsAWSChunked(%v) = %v, want %v", tt.headers, got, tt.want)
			}
		})
	}
	if got := StoredContentEncoding(chunkedRequest(nil, map[string]string{"Content-Encoding": "aws-chunked,gzip"})); got != "gzip" {
		t.Fatalf("StoredContentEncoding = %q, want gzip", got)
	}
}
