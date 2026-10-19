package aws

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// aws-chunked request bodies.
//
// AWS SDKs send PutObject bodies in the aws-chunked framing whenever they add a
// streaming signature or a trailing checksum, and SDK v2 (Go, JS v3, Java 2,
// boto3 >= 1.36) does the latter by default. The body then looks like
//
//	<hex size>[;chunk-signature=<sig>]\r\n<data>\r\n ... 0[;chunk-signature=<sig>]\r\n[<trailer>: <value>\r\n...]\r\n
//
// Storing that verbatim corrupts the object (Hillview, 2026-10-18), so the
// upload handler decodes it with NewChunkedReader before writing.
//
// Chunk signatures are parsed but not verified: the seed signature over the
// headers is checked as usual, and integrity comes from the trailing checksum
// (verified here) and Content-MD5 (verified by objects.CreateObjectVerified).

// ErrChunkedFraming means the body claimed aws-chunked but wasn't well formed.
var ErrChunkedFraming = errors.New("malformed aws-chunked body")

// ErrChunkedChecksum means a trailing x-amz-checksum-* didn't match the data.
var ErrChunkedChecksum = errors.New("aws-chunked trailing checksum mismatch")

// ErrChunkedLength means the decoded body didn't match x-amz-decoded-content-length.
var ErrChunkedLength = errors.New("aws-chunked decoded length mismatch")

// maxChunkLine bounds a chunk header or trailer line.
const maxChunkLine = 4096

// IsAWSChunked reports whether the request body uses the aws-chunked framing.
func IsAWSChunked(r *http.Request) bool {
	if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
		return true
	}
	for _, enc := range strings.Split(r.Header.Get("Content-Encoding"), ",") {
		if strings.EqualFold(strings.TrimSpace(enc), "aws-chunked") {
			return true
		}
	}
	return false
}

// StoredContentEncoding is the Content-Encoding to keep for the object: the
// request's, minus the aws-chunked transfer framing.
func StoredContentEncoding(r *http.Request) string {
	var keep []string
	for _, enc := range strings.Split(r.Header.Get("Content-Encoding"), ",") {
		enc = strings.TrimSpace(enc)
		if enc != "" && !strings.EqualFold(enc, "aws-chunked") {
			keep = append(keep, enc)
		}
	}
	return strings.Join(keep, ",")
}

type chunkedReader struct {
	br        *bufio.Reader
	remaining int64 // bytes left in the current chunk
	done      bool
	err       error

	decoded     int64
	wantDecoded int64 // -1 when not declared

	checksumHeader string // e.g. "x-amz-checksum-crc32", "" when no trailer declared
	sum            hash.Hash
}

// NewChunkedReader decodes an aws-chunked request body. It verifies the
// trailing checksum named by X-Amz-Trailer (crc32, crc32c, sha256) and the
// length in X-Amz-Decoded-Content-Length when present; a failure surfaces as an
// error from Read at the end of the body.
func NewChunkedReader(r *http.Request) io.Reader {
	cr := &chunkedReader{br: bufio.NewReaderSize(r.Body, 64*1024), wantDecoded: -1}
	if v := r.Header.Get("X-Amz-Decoded-Content-Length"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			cr.wantDecoded = n
		}
	}
	if t := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Amz-Trailer"))); t != "" {
		cr.checksumHeader = t
		cr.sum = checksumHash(t)
	}
	return cr
}

func checksumHash(name string) hash.Hash {
	switch name {
	case "x-amz-checksum-crc32":
		return crc32.NewIEEE()
	case "x-amz-checksum-crc32c":
		return crc32.New(crc32.MakeTable(crc32.Castagnoli))
	case "x-amz-checksum-sha256":
		return sha256.New()
	}
	return nil // sha1, crc64nvme, unknown: decoded, not verified
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	for c.remaining == 0 {
		if c.done {
			return 0, io.EOF
		}
		if err := c.nextChunk(); err != nil {
			c.err = err
			return 0, err
		}
	}
	if int64(len(p)) > c.remaining {
		p = p[:c.remaining]
	}
	n, err := c.br.Read(p)
	c.remaining -= int64(n)
	c.decoded += int64(n)
	if c.sum != nil {
		_, _ = c.sum.Write(p[:n]) // hash.Hash.Write never fails
	}
	if err == io.EOF {
		err = fmt.Errorf("%w: body ended inside a chunk", ErrChunkedFraming)
	}
	if err != nil {
		c.err = err
		return n, err
	}
	if c.remaining == 0 {
		if err := c.expectCRLF(); err != nil {
			c.err = err
			return n, err
		}
	}
	return n, nil
}

// nextChunk reads a chunk header; on the final (zero) chunk it consumes the
// trailers and runs the end-of-body checks.
func (c *chunkedReader) nextChunk() error {
	line, err := c.readLine()
	if err != nil {
		return err
	}
	sizeField, _, _ := strings.Cut(line, ";")
	size, err := strconv.ParseInt(strings.TrimSpace(sizeField), 16, 64)
	if err != nil || size < 0 {
		return fmt.Errorf("%w: bad chunk size %q", ErrChunkedFraming, sizeField)
	}
	if size > 0 {
		c.remaining = size
		return nil
	}
	c.done = true
	return c.finish()
}

func (c *chunkedReader) finish() error {
	trailers := map[string]string{}
	for {
		line, err := c.readLine()
		if errors.Is(err, io.EOF) && len(trailers) == 0 {
			break // some clients end the body right after "0\r\n"
		}
		if err != nil {
			return err
		}
		if line == "" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return fmt.Errorf("%w: bad trailer line", ErrChunkedFraming)
		}
		trailers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	if c.wantDecoded >= 0 && c.decoded != c.wantDecoded {
		return fmt.Errorf("%w: got %d bytes, header said %d", ErrChunkedLength, c.decoded, c.wantDecoded)
	}
	if c.checksumHeader != "" && c.sum != nil {
		got, ok := trailers[c.checksumHeader]
		if !ok {
			return fmt.Errorf("%w: trailer %s missing", ErrChunkedChecksum, c.checksumHeader)
		}
		if want := base64.StdEncoding.EncodeToString(c.sum.Sum(nil)); got != want {
			return fmt.Errorf("%w: %s", ErrChunkedChecksum, c.checksumHeader)
		}
	}
	return nil
}

func (c *chunkedReader) readLine() (string, error) {
	line, err := c.br.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) || len(line) > maxChunkLine {
		return "", fmt.Errorf("%w: line too long", ErrChunkedFraming)
	}
	if err != nil {
		if errors.Is(err, io.EOF) && len(line) == 0 {
			return "", io.EOF
		}
		return "", fmt.Errorf("%w: %v", ErrChunkedFraming, err)
	}
	return string(bytes.TrimRight(line, "\r\n")), nil
}

func (c *chunkedReader) expectCRLF() error {
	var b [2]byte
	if _, err := io.ReadFull(c.br, b[:]); err != nil || b != [2]byte{'\r', '\n'} {
		return fmt.Errorf("%w: chunk not followed by CRLF", ErrChunkedFraming)
	}
	return nil
}
