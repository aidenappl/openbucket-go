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

func GeneratePresignedURL(bucket string, key string, expirationTime int64) string {
	var secretKey = "your-secret-key"

	now := time.Now().Unix()

	expiration := now + expirationTime

	stringToSign := fmt.Sprintf("%s/%s/%s?expires=%d", bucket, key, secretKey, expiration)

	hmacSignature := generateHMACSignature(stringToSign, secretKey)

	url := fmt.Sprintf("http://localhost:8080/%s/%s?expires=%d&signature=%s", bucket, key, expiration, hmacSignature)

	return url
}

func generateHMACSignature(data string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func ValidatePresignedURL(secretKey string, r *http.Request) bool {

	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	expiration := r.URL.Query().Get("expires")
	signature := r.URL.Query().Get("signature")

	if bucket == "" || key == "" || expiration == "" || signature == "" {
		return false
	}

	expirationTime, err := strconv.ParseInt(expiration, 10, 64)
	if err != nil {
		return false
	}

	currentTime := time.Now().Unix()
	if currentTime > expirationTime {
		return false
	}

	stringToSign := fmt.Sprintf("%s/%s/%s?expires=%s", bucket, key, secretKey, expiration)
	expectedSignature := generateHMACSignature(stringToSign, secretKey)

	if expectedSignature != signature {
		return false
	}

	return true
}
