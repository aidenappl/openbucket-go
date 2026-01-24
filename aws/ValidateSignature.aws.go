package aws

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
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
// Returns true if the signature is valid, false otherwise.
func ValidateSignature(r *http.Request, authorizationHeader, dateHeader, amzContentSHA256 string) bool {

	parts := strings.Split(authorizationHeader, " ")
	if parts[0] != "AWS4-HMAC-SHA256" {
		log.Println("Invalid Authorization header format")
		return false
	}

	credentialParts := strings.Split(parts[1], "=")
	if len(credentialParts) != 2 || credentialParts[0] != "Credential" {
		log.Println("Invalid Credential format in Authorization header:", credentialParts)
		return false
	}

	accessKey := strings.Split(credentialParts[1], "/")[0]
	if accessKey == "" {
		log.Println("Access Key is missing in Authorization header")
		return false
	}

	signedHeadersParts := strings.Split(parts[2], "=")
	if len(signedHeadersParts) != 2 || signedHeadersParts[0] != "SignedHeaders" {
		log.Println("Invalid SignedHeaders format in Authorization header:", signedHeadersParts)
		return false
	}
	rawSH := signedHeadersParts[1]
	rawSH = strings.TrimSpace(rawSH)
	rawSH = strings.TrimSuffix(rawSH, ",")

	signatureParts := strings.Split(parts[3], "=")
	if signatureParts[0] != "Signature" {
		log.Println("Invalid Signature format in Authorization header:", signatureParts)
		return false
	}

	signature := signatureParts[1]
	if signature == "" {
		log.Println("Signature is missing in Authorization header")
		return false
	}

	date, err := time.Parse("20060102T150405Z", dateHeader)
	if err != nil {
		log.Println("Error parsing date:", dateHeader, err)
		return false
	}

	secretKey, err := loadSecretKeyByAccessKey(accessKey)
	if err != nil {
		log.Println("Error loading secret key for Access Key:", accessKey, err)
		return false
	}
	if secretKey == nil {
		log.Println("Secret Key not found for Access Key:", accessKey)
		return false
	}

	// Debug: log first/last 4 chars of secret key to verify it's correct
	sk := *secretKey
	if len(sk) > 8 {
		log.Printf("DEBUG: Using secret key: %s...%s (length: %d)", sk[:4], sk[len(sk)-4:], len(sk))
	}

	canonicalRequest := buildCanonicalRequest(r, rawSH, amzContentSHA256)

	stringToSign := buildStringToSign(date, env.Region, "s3", canonicalRequest)

	signingKey := getSigningKey(*secretKey, date, env.Region, "s3")

	computedSignature := computeSignature(signingKey, stringToSign)

	if computedSignature != signature {
		log.Println("Signature mismatch: computed signature does not match header signature")
		log.Printf("Canonical Request:\n%s\n", canonicalRequest)
		log.Printf("String to Sign:\n%s\n", stringToSign)
		log.Printf("Computed: %s", computedSignature)
		log.Printf("Received: %s", signature)
		// Debug: dump canonical request as Go string literal to see exact bytes
		log.Printf("DEBUG: Canonical request bytes: %q", canonicalRequest)
		// Debug: show signing key derivation inputs
		log.Printf("DEBUG: Signing key inputs - date: %s, region: %s, service: s3", date.Format("20060102"), env.Region)
		log.Printf("DEBUG: Signing key (first 8 bytes hex): %x", signingKey[:8])
		// Debug: dump all relevant request details
		log.Printf("DEBUG: Signed headers from auth: %s", rawSH)
		log.Printf("DEBUG: Content-Length header: %s, r.ContentLength: %d", r.Header.Get("Content-Length"), r.ContentLength)
		log.Printf("DEBUG: All request headers:")
		for name, values := range r.Header {
			log.Printf("  %s: %v", name, values)
		}
		return false
	}

	return true
}
func buildCanonicalRequest(r *http.Request,
	signedHeadersCSV, payloadHash string) string {

	if r.Header.Get("Host") == "" {
		r.Header.Set("Host", r.Host)
	}

	hdrNames := strings.Split(signedHeadersCSV, ";")
	var clean []string
	for _, h := range hdrNames {
		h = strings.TrimSpace(h)
		if h != "" {
			clean = append(clean, h)
		}
	}
	sort.Strings(clean)

	var canon strings.Builder
	for _, h := range clean {
		var v string

		// Handle special headers that Go's HTTP library parses specially
		switch strings.ToLower(h) {
		case "content-length":
			// Go parses Content-Length into r.ContentLength, not r.Header
			if r.ContentLength > 0 {
				v = strconv.FormatInt(r.ContentLength, 10)
			} else if val := r.Header.Get(h); val != "" {
				v = strings.TrimSpace(stripExcessSpaces(val))
			}
		case "host":
			// Host may be in r.Host instead of r.Header
			if hostVal := r.Header.Get("Host"); hostVal != "" {
				v = strings.TrimSpace(stripExcessSpaces(hostVal))
			} else {
				v = r.Host
			}
		default:
			values := r.Header.Values(h)
			var cleanedValues []string
			for _, val := range values {
				// Trim leading/trailing spaces and collapse multiple consecutive spaces
				cleanedValues = append(cleanedValues, strings.TrimSpace(stripExcessSpaces(val)))
			}
			v = strings.Join(cleanedValues, ",")
		}

		canon.WriteString(strings.ToLower(h))
		canon.WriteString(":")
		canon.WriteString(v)
		canon.WriteString("\n")
	}

	// Use EscapedPath to get the encoded path, then normalize it
	uri := r.URL.EscapedPath()
	if uri == "" {
		uri = "/"
	}
	// Decode and re-encode to normalize according to AWS rules
	decodedPath, _ := url.PathUnescape(uri)
	uri = canonicalURI(decodedPath)

	query := canonicalQueryFromRaw(r.URL.RawQuery)

	// AWS Canonical Request format:
	// HTTPMethod\n
	// CanonicalURI\n
	// CanonicalQueryString\n
	// CanonicalHeaders\n  <- canon.String() already ends with \n
	// SignedHeaders\n
	// HashedPayload
	return fmt.Sprintf("%s\n%s\n%s\n%s%s\n%s",
		r.Method,
		uri,
		query,
		canon.String(), // already ends with \n, so no \n before next %s
		strings.Join(clean, ";"),
		payloadHash,
	)
}

// canonicalURI encodes a URI path according to AWS Signature V4 rules.
// Each path segment is encoded, but '/' delimiters are preserved.
func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}

	// Split path into segments and encode each one
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = awsURIEncode(segment, false)
	}

	result := strings.Join(segments, "/")

	// Ensure path starts with /
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	return result
}

// canonicalQueryFromRaw builds a canonical query string from raw query parameters.
// It decodes, re-encodes according to AWS rules, and sorts parameters.
// Certain SDK-internal parameters like x-id are excluded as they are not signed.
func canonicalQueryFromRaw(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	// Parameters that AWS SDK adds after signing (not included in signature)
	unsignedParams := map[string]bool{
		"x-id": true,
		"X-id": true,
		"X-Id": true,
		"X-ID": true,
	}

	// Parse the raw query string manually
	params := make(map[string][]string)
	pairs := strings.Split(rawQuery, "&")

	for _, pair := range pairs {
		if pair == "" {
			continue
		}

		kv := strings.SplitN(pair, "=", 2)
		key := kv[0]
		val := ""
		if len(kv) == 2 {
			val = kv[1]
		}

		// Decode the key and value to get the actual values
		decodedKey, _ := url.QueryUnescape(key)
		decodedVal, _ := url.QueryUnescape(val)

		// Skip unsigned SDK-internal parameters
		if unsignedParams[decodedKey] {
			continue
		}

		// Re-encode using AWS rules (encode slashes in query params)
		encodedKey := awsURIEncode(decodedKey, true)
		encodedVal := awsURIEncode(decodedVal, true)

		params[encodedKey] = append(params[encodedKey], encodedVal)
	}

	// Sort and build canonical query string
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vals := params[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, k+"="+v)
		}
	}

	return strings.Join(parts, "&")
}

// awsURIEncode encodes a string according to AWS Signature V4 URI encoding rules.
// Unreserved characters (A-Z, a-z, 0-9, '-', '_', '.', '~') are not encoded.
// Set encodePath to true to encode '/' (for query parameters).
// Set encodePath to false to preserve '/' (for URI paths).
func awsURIEncode(s string, encodePath bool) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '-' || c == '~' || c == '.' {
			result.WriteByte(c)
		} else if c == '/' && !encodePath {
			result.WriteByte(c)
		} else {
			result.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return result.String()
}

// stripExcessSpaces collapses multiple consecutive spaces into a single space
// and trims leading/trailing spaces. This matches AWS SDK behavior for
// canonicalizing header values. Tabs are converted to spaces first.
func stripExcessSpaces(str string) string {
	// First, convert all tabs to spaces (AWS SDK behavior)
	str = strings.ReplaceAll(str, "\t", " ")

	var j, k, l, m, spaces int
	// Trim trailing spaces
	for j = len(str) - 1; j >= 0 && str[j] == ' '; j-- {
	}

	// Trim leading spaces
	for k = 0; k < j && str[k] == ' '; k++ {
	}
	if k > j {
		return ""
	}
	str = str[k : j+1]

	// Strip multiple spaces
	idx := strings.Index(str, "  ")
	if idx < 0 {
		return str
	}

	buf := []byte(str)
	for k, m, l = idx, idx, len(buf); k < l; k++ {
		if buf[k] == ' ' {
			if spaces == 0 {
				// First space
				buf[m] = buf[k]
				m++
			}
			spaces++
		} else {
			// End of multiple spaces
			spaces = 0
			buf[m] = buf[k]
			m++
		}
	}

	return string(buf[:m])
}

func buildStringToSign(date time.Time, region, service, canonicalRequest string) string {
	dateStr := date.Format("20060102T150405Z")
	scope := fmt.Sprintf("%s/%s/%s/aws4_request", date.Format("20060102"), region, service)

	hash := sha256.New()
	hash.Write([]byte(canonicalRequest))
	canonicalRequestHash := fmt.Sprintf("%x", hash.Sum(nil))

	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", dateStr, scope, canonicalRequestHash)
}

func getSigningKey(secret string, date time.Time, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date.Format("20060102"))
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	return kSigning
}

func computeSignature(signingKey []byte, stringToSign string) string {
	return fmt.Sprintf("%x", hmacSHA256(signingKey, stringToSign))
}

func hmacSHA256(key []byte, message string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return h.Sum(nil)
}

func loadSecretKeyByAccessKey(accessKey string) (*string, error) {
	query, err := db.Psql.Select(
		"secret_key",
	).From("authorizations").Where(sq.Eq{"key_id": accessKey}).Query()
	if err != nil {
		return nil, fmt.Errorf("error querying secret key: %w", err)
	}
	defer query.Close()

	var secretKey string
	if query.Next() {
		if err := query.Scan(&secretKey); err != nil {
			return nil, fmt.Errorf("error scanning secret key: %w", err)
		}
		return &secretKey, nil
	}

	return nil, fmt.Errorf("secret key not found for access key: %s", accessKey)
}
