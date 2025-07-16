package aws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

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

	canonicalRequest := buildCanonicalRequest(r, rawSH, amzContentSHA256)

	stringToSign := buildStringToSign(date, "garage", "s3", canonicalRequest)

	signingKey := getSigningKey(secretKey, date, "garage", "s3")

	computedSignature := computeSignature(signingKey, stringToSign)

	if computedSignature != signature {
		log.Println("Signature mismatch: computed signature does not match header signature")
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
		v := strings.Join(r.Header.Values(h), ",")
		canon.WriteString(strings.ToLower(h))
		canon.WriteString(":")
		canon.WriteString(strings.TrimSpace(v))
		canon.WriteString("\n")
	}

	uri := r.URL.EscapedPath()
	if uri == "" {
		uri = "/"
	}
	query := canonicalQuery(r.URL.Query())

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		r.Method,
		uri,
		query,
		canon.String(),
		strings.Join(clean, ";"),
		payloadHash,
	)
}

func canonicalQuery(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	var parts []string
	for k, vs := range v {
		ek := url.QueryEscape(k)
		sort.Strings(vs)
		for _, val := range vs {
			parts = append(parts, ek+"="+url.QueryEscape(val))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
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

func loadSecretKeyByAccessKey(accessKey string) (string, error) {
	file, err := os.Open("authorizations.xml")
	if err != nil {
		return "", fmt.Errorf("error opening Authorizations file: %w", err)
	}
	defer file.Close()

	var authorizations struct {
		Authorizations []struct {
			AccessKeyID string `xml:"KEY_ID"`
			SecretKey   string `xml:"SECRET_KEY"`
		} `xml:"Authorization"`
	}

	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&authorizations); err != nil {
		return "", fmt.Errorf("error parsing Authorizations XML: %w", err)
	}

	for _, authorization := range authorizations.Authorizations {
		if authorization.AccessKeyID == accessKey {
			return authorization.SecretKey, nil
		}
	}

	return "", fmt.Errorf("secret key not found for access key: %s", accessKey)
}
