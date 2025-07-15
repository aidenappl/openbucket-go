package tools

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// GeneratePresignedURL generates a custom signed URL
func GeneratePresignedURL(bucket string, key string, expirationTime int64) string {
	var secretKey = "your-secret-key" // Replace with your actual secret key

	// Current timestamp
	now := time.Now().Unix()
	// Expiration time (in seconds)
	expiration := now + expirationTime

	// URL to be signed (simplified)
	stringToSign := fmt.Sprintf("%s/%s/%s?expires=%d", bucket, key, secretKey, expiration)

	// Generate the HMAC-SHA256 signature
	hmacSignature := generateHMACSignature(stringToSign, secretKey)

	// Construct the final presigned URL with the expiration and signature
	url := fmt.Sprintf("http://localhost:8080/%s/%s?expires=%d&signature=%s", bucket, key, expiration, hmacSignature)

	return url
}

// generateHMACSignature generates an HMAC-SHA256 signature for a given string
func generateHMACSignature(data string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ValidatePresignedURL checks if the presigned URL is valid and hasn't expired
func ValidatePresignedURL(secretKey string, r *http.Request) bool {
	// Extract the bucket, key, expiration, and signature from the URL query parameters
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	expiration := r.URL.Query().Get("expires")
	signature := r.URL.Query().Get("signature")

	// Ensure the required parameters are present
	if bucket == "" || key == "" || expiration == "" || signature == "" {
		return false
	}

	// Parse the expiration timestamp to int64
	expirationTime, err := strconv.ParseInt(expiration, 10, 64)
	if err != nil {
		return false
	}

	// Check if the URL has expired
	currentTime := time.Now().Unix()
	if currentTime > expirationTime {
		return false // URL has expired
	}

	// Regenerate the HMAC-SHA256 signature from the same data
	stringToSign := fmt.Sprintf("%s/%s/%s?expires=%s", bucket, key, secretKey, expiration)
	expectedSignature := generateHMACSignature(stringToSign, secretKey)

	// Compare the expected signature with the provided signature
	if expectedSignature != signature {
		return false // Signatures do not match
	}

	return true // Valid presigned URL
}
