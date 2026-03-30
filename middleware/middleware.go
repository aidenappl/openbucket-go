package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := GetRequestID(r)
		host := GetHostID(r)
		log.Println(r.Method, r.RequestURI, fmt.Sprintf("|| Request ID: %s", request), fmt.Sprintf("Host: %s", host))
		next.ServeHTTP(w, r)
	})
}

// isFileDownload checks if the request is for a file download (not an API operation)
func isFileDownload(r *http.Request) bool {
	// Must be GET request
	if r.Method != http.MethodGet {
		return false
	}

	// Must have at least bucket/key pattern (more than just /bucket)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		return false
	}

	// API operations use query parameters like ?acl, ?tagging, ?location, etc.
	q := r.URL.Query()
	apiParams := []string{"acl", "tagging", "location", "policy", "versioning",
		"uploads", "versions", "lifecycle", "cors", "encryption", "attributes"}
	for _, param := range apiParams {
		if q.Has(param) {
			return false
		}
	}

	return true
}

func MuxHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, "+
			"Content-Type, "+
			"Accept-Encoding, "+
			"Connection, "+
			"Content-Length")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Server", "Go")

		// Allow caching for file downloads, prevent caching for API responses
		if !isFileDownload(r) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("CDN-Cache-Control", "no-store")
			w.Header().Set("Cloudflare-CDN-Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}
