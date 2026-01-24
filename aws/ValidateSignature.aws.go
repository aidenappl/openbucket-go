package aws

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/env"
)

// ValidateSignature validates an AWS Signature V4 request signature.
// Implementation matches aws-sdk-go-v2/aws/signer/v4 exactly.
func ValidateSignature(r *http.Request, authorizationHeader, dateHeader, amzContentSHA256 string) bool {
	// Parse Authorization header: AWS4-HMAC-SHA256 Credential=.../..., SignedHeaders=..., Signature=...
	if !strings.HasPrefix(authorizationHeader, "AWS4-HMAC-SHA256 ") {
		return false
	}

	authValue := strings.TrimPrefix(authorizationHeader, "AWS4-HMAC-SHA256 ")
	parts := strings.Split(authValue, ", ")
	if len(parts) != 3 {
		return false
	}

	// Parse Credential
	if !strings.HasPrefix(parts[0], "Credential=") {
		return false
	}
	credential := strings.TrimPrefix(parts[0], "Credential=")
	credParts := strings.Split(credential, "/")
	if len(credParts) < 5 {
		return false
	}
	accessKey := credParts[0]

	// Parse SignedHeaders
	if !strings.HasPrefix(parts[1], "SignedHeaders=") {
		return false
	}
	signedHeaders := strings.TrimPrefix(parts[1], "SignedHeaders=")

	// Parse Signature
	if !strings.HasPrefix(parts[2], "Signature=") {
		return false
	}
	signature := strings.TrimPrefix(parts[2], "Signature=")

	// Parse date
	date, err := time.Parse("20060102T150405Z", dateHeader)
	if err != nil {
		return false
	}

	// Load secret key
	secretKey, err := loadSecretKeyByAccessKey(accessKey)
	if err != nil || secretKey == nil {
		return false
	}

	// Build canonical request and compute signature
	canonicalRequest := buildCanonicalRequest(r, signedHeaders, amzContentSHA256)
	stringToSign := buildStringToSign(date, env.Region, "s3", canonicalRequest)
	signingKey := deriveSigningKey(*secretKey, date, env.Region, "s3")
	computedSignature := computeSignature(signingKey, stringToSign)

	return computedSignature == signature
}

// buildCanonicalRequest builds the canonical request string per AWS Signature V4 spec.
// Reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
func buildCanonicalRequest(r *http.Request, signedHeaders, payloadHash string) string {
	// Parse and sort signed headers
	headers := strings.Split(signedHeaders, ";")
	sort.Strings(headers)

	// Build canonical headers
	var canonicalHeaders strings.Builder
	for _, h := range headers {
		h = strings.ToLower(h)
		var value string

		switch h {
		case "host":
			value = r.Host
			if v := r.Header.Get("Host"); v != "" {
				value = v
			}
		case "content-length":
			if r.ContentLength >= 0 {
				value = strconv.FormatInt(r.ContentLength, 10)
			} else if v := r.Header.Get("Content-Length"); v != "" {
				value = v
			}
		default:
			// Get all values for this header and join with comma
			values := r.Header.Values(h)
			var trimmed []string
			for _, v := range values {
				trimmed = append(trimmed, trimHeaderValue(v))
			}
			value = strings.Join(trimmed, ",")
		}

		canonicalHeaders.WriteString(h)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(trimHeaderValue(value))
		canonicalHeaders.WriteString("\n")
	}

	// Build canonical URI (path)
	uri := canonicalURI(r.URL.Path)

	// Build canonical query string
	query := canonicalQueryString(r.URL.RawQuery)

	// Canonical request format:
	// HTTPMethod\n
	// CanonicalURI\n
	// CanonicalQueryString\n
	// CanonicalHeaders\n
	// SignedHeaders\n
	// HashedPayload
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		r.Method,
		uri,
		query,
		canonicalHeaders.String(),
		strings.Join(headers, ";"),
		payloadHash,
	)
}

// canonicalURI returns the URI-encoded version of the path.
// Reference: aws-sdk-go-v2/aws/signer/v4/v4.go
func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}

	// Double-encode except for forward slashes
	// First decode to handle any existing encoding
	decoded, err := url.PathUnescape(path)
	if err != nil {
		decoded = path
	}

	// Encode each segment
	segments := strings.Split(decoded, "/")
	for i, seg := range segments {
		segments[i] = uriEncode(seg, false)
	}

	encoded := strings.Join(segments, "/")
	if !strings.HasPrefix(encoded, "/") {
		encoded = "/" + encoded
	}

	return encoded
}

// canonicalQueryString builds the canonical query string.
// Parameters are sorted by key name, then by value.
// Reference: aws-sdk-go-v2/aws/signer/v4/v4.go
func canonicalQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	params := make(map[string][]string)
	pairs := strings.Split(rawQuery, "&")

	for _, pair := range pairs {
		if pair == "" {
			continue
		}

		idx := strings.Index(pair, "=")
		var key, value string
		if idx == -1 {
			key = pair
			value = ""
		} else {
			key = pair[:idx]
			value = pair[idx+1:]
		}

		decodedKey, _ := url.QueryUnescape(key)
		decodedValue, _ := url.QueryUnescape(value)

		encodedKey := uriEncode(decodedKey, true)
		encodedValue := uriEncode(decodedValue, true)

		params[encodedKey] = append(params[encodedKey], encodedValue)
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		values := params[k]
		sort.Strings(values)
		for _, v := range values {
			parts = append(parts, k+"="+v)
		}
	}

	return strings.Join(parts, "&")
}

// uriEncode encodes a string per AWS Signature V4 URI encoding rules.
// Unreserved characters: A-Z a-z 0-9 - _ . ~
// Reference: aws-sdk-go-v2/aws/signer/v4/uri.go
func uriEncode(s string, encodeSlash bool) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) || (c == '/' && !encodeSlash) {
			buf.WriteByte(c)
		} else {
			fmt.Fprintf(&buf, "%%%02X", c)
		}
	}
	return buf.String()
}

// isUnreserved returns true if the character is an unreserved character per RFC 3986.
func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}

// trimHeaderValue trims leading/trailing whitespace and collapses
// multiple consecutive spaces into a single space.
// Reference: aws-sdk-go-v2/aws/signer/internal/v4/util.go StripExcessSpaces
func trimHeaderValue(s string) string {
	var j, k, l, m, spaces int

	// Trim trailing spaces
	for j = len(s) - 1; j >= 0 && s[j] == ' '; j-- {
	}

	// Trim leading spaces
	for k = 0; k < j && s[k] == ' '; k++ {
	}

	if k > j {
		return ""
	}

	s = s[k : j+1]

	// Strip multiple spaces
	idx := strings.Index(s, "  ")
	if idx < 0 {
		return s
	}

	buf := []byte(s)
	for k, m, l = idx, idx, len(buf); k < l; k++ {
		if buf[k] == ' ' {
			if spaces == 0 {
				buf[m] = buf[k]
				m++
			}
			spaces++
		} else {
			spaces = 0
			buf[m] = buf[k]
			m++
		}
	}

	return string(buf[:m])
}

// buildStringToSign creates the string to sign for AWS Signature V4.
// Reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-create-string-to-sign.html
func buildStringToSign(date time.Time, region, service, canonicalRequest string) string {
	scope := fmt.Sprintf("%s/%s/%s/aws4_request",
		date.Format("20060102"), region, service)

	hash := sha256.Sum256([]byte(canonicalRequest))

	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%x",
		date.Format("20060102T150405Z"),
		scope,
		hash,
	)
}

// deriveSigningKey derives the signing key using HMAC-SHA256.
// Reference: https://docs.aws.amazon.com/general/latest/gr/sigv4-calculate-signature.html
func deriveSigningKey(secret string, date time.Time, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date.Format("20060102"))
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

// computeSignature computes the final signature.
func computeSignature(signingKey []byte, stringToSign string) string {
	return fmt.Sprintf("%x", hmacSHA256(signingKey, stringToSign))
}

// hmacSHA256 computes HMAC-SHA256.
func hmacSHA256(key []byte, message string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return h.Sum(nil)
}

// loadSecretKeyByAccessKey retrieves the secret key for the given access key.
func loadSecretKeyByAccessKey(accessKey string) (*string, error) {
	query, err := db.Psql.Select("secret_key").
		From("authorizations").
		Where(sq.Eq{"key_id": accessKey}).
		Query()
	if err != nil {
		return nil, err
	}
	defer query.Close()

	if query.Next() {
		var secretKey string
		if err := query.Scan(&secretKey); err != nil {
			return nil, err
		}
		return &secretKey, nil
	}

	return nil, nil
}
