package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/aws"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

type contextKey string

var BucketContextKey contextKey = "bucket"
var MetadataContextKey contextKey = "metadata"
var SessionContextKey contextKey = "session"

// HalfAuthorized is a middleware that only checks if the user is authorized to access the endpoint
func HalfAuthorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		requestID, hostID := GetRequestID(r), GetHostID(r)

		// deny handles general access denial with logging
		deny := func(msg string, err error) {
			responder.SendAccessDeniedXML(w, &requestID, &hostID)
			if err != nil {
				log.Printf("%s: %v", msg, err)
			} else {
				log.Println(msg)
			}
		}

		// Get the access key from the request
		keyID, err := GetAccessKeyFromRequest(r)
		if err != nil {
			deny("Unauthorized: missing or invalid access key", err)
			return
		}

		// Check if the user exists
		session, err := auth.CheckUserExists(keyID)
		if err != nil {
			deny("Unauthorized: "+err.Error(), nil)
			return
		}
		if session == nil {
			deny(fmt.Sprintf("Unauthorized: user with KEY_ID %s not found", keyID), nil)
			return
		}

		// Validate AWS signature to prevent access with only a valid key ID
		if !validateAWSSignature(r) {
			deny("Invalid AWS signature for "+r.Method+" "+r.URL.Path, nil)
			return
		}

		vars := mux.Vars(r)
		bucketName := vars["bucket"]

		if bucketName != "" {
			// Get the permissions for the bucket
			perms, err := bucket.GetBucket(bucketName)
			if err != nil {
				deny("Error loading permissions for bucket "+bucketName, err)
				return
			}
			if perms != nil {
				ctx = context.WithValue(ctx, BucketContextKey, perms)
			}

		}

		// Store the session in the context
		ctx = context.WithValue(ctx, SessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// Authorized is a middleware that checks if the user is authorized to access the requested resource.
func Authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return 404 for favicon and robots.txt
		if strings.HasSuffix(r.URL.Path, "favicon.ico") || strings.HasSuffix(r.URL.Path, "robots.txt") {
			http.NotFound(w, r)
			return
		}

		vars := mux.Vars(r)
		bucketName, key := vars["bucket"], vars["key"]
		requestID, hostID := GetRequestID(r), GetHostID(r)
		ctx := r.Context()

		// deny handles general access denial with logging
		deny := func(msg string, err error) {
			responder.SendAccessDeniedXML(w, &requestID, &hostID)
			if err != nil {
				log.Printf("%s: %v", msg, err)
			} else {
				log.Println(msg)
			}
		}

		// Get the permissions for the bucket
		perms, err := bucket.GetBucket(bucketName)
		if err != nil {
			deny("Error loading permissions for bucket "+bucketName, err)
			return
		}
		if perms != nil {
			ctx = context.WithValue(ctx, BucketContextKey, perms)
		}

		// Load object metadata
		obj, err := objects.GetObject(bucketName, key, nil)
		if err != nil {
			deny("Error loading object metadata for "+key, err)
			return
		}
		if obj != nil {
			ctx = context.WithValue(ctx, MetadataContextKey, obj)
		}

		// Do a fast path check for public access or ACL permissions
		if isFastPathAllowed(perms, ctx, r) {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Check if requesting object for download
		if obj != nil && r.URL.Path == fmt.Sprintf("/%s/%s", bucketName, key) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			// Allow public access to the object if it is marked as public
			if obj.Public {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Validate AWS signature if bypass is not enabled
		if !validateAWSSignature(r) {
			deny("Invalid AWS signature for "+r.Method+" "+r.URL.Path, nil)
			return
		}

		// Extract the access key from the request
		keyID, err := GetAccessKeyFromRequest(r)
		if err != nil {
			deny("Unauthorized: missing or invalid access key", nil)
			return
		}

		// Check if the user exists
		if bucketName == "" {
			session, err := auth.CheckUserExists(keyID)
			if err != nil {
				deny("Unauthorized: "+err.Error(), nil)
				return
			}
			ctx = context.WithValue(ctx, SessionContextKey, session)
		}

		// Authorize against the bucket ACL
		session, err := authoriseByACL(keyID, bucketName, r)
		if err != nil {
			deny("Forbidden: "+err.Error(), nil)
			return
		}
		// Store the session in the context
		ctx = context.WithValue(ctx, SessionContextKey, session)

		// serve the request with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// isFastPathAllowed checks if the request can be served without further permission checks.
// Write and delete operations always require authentication and are never fast-pathed.
func isFastPathAllowed(perms *types.Bucket, ctx context.Context, r *http.Request) bool {
	// Never skip auth for write/delete operations
	if isWriteRoute(r) {
		return false
	}

	if perms != nil {
		if isReadRoute(r) && types.IsBucketACLRead(perms.ACL) {
			return true
		}
	}

	// Object-level public flag (read only)
	if mdRaw := r.Context().Value(MetadataContextKey); mdRaw != nil && isReadRoute(r) {
		if md, ok := mdRaw.(*types.ObjectMetadata); ok && md.Public {
			return true
		}
	}

	return false
}

// authoriseByACL checks the user's permissions against the bucket ACL
func authoriseByACL(keyID, bucket string, r *http.Request) (*types.Authorization, error) {
	session, err := auth.CheckUserExists(keyID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("user with KEY_ID %s not found", keyID)
	}

	userACL, err := auth.CheckUserPermissions(keyID, bucket)
	if err != nil {
		return nil, err
	}
	if userACL == nil {
		return nil, fmt.Errorf("user %s has no ACL for bucket %s", keyID, bucket)
	}

	switch {
	case isWriteRoute(r) && !types.IsWritePermission(userACL.Permission):
		return nil, fmt.Errorf("user %s lacks write permission on bucket %s", keyID, bucket)
	case isReadRoute(r) && !types.IsReadPermission(userACL.Permission):
		return nil, fmt.Errorf("user %s lacks read permission on bucket %s", keyID, bucket)
	}
	return session, nil
}

// GetAccessKeyFromRequest extracts the access key from the request's Authorization header.
func GetAccessKeyFromRequest(r *http.Request) (string, error) {

	// Check for presigned URL credential first
	credential := r.URL.Query().Get("X-Amz-Credential")
	if credential != "" {
		// Credential format: ACCESS_KEY/DATE/REGION/SERVICE/aws4_request
		accessKey := strings.Split(credential, "/")[0]
		if accessKey != "" {
			return accessKey, nil
		}
	}

	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	parts := strings.Split(authorizationHeader, " ")
	if parts[0] != "AWS4-HMAC-SHA256" {
		log.Println("Invalid Authorization header format for AWS signature")
		return "", fmt.Errorf("invalid Authorization header format for AWS signature")
	}

	credentialParts := strings.Split(parts[1], "=")
	if len(credentialParts) != 2 || credentialParts[0] != "Credential" {
		log.Println("Invalid Credential format in Authorization header:", credentialParts)
		return "", fmt.Errorf("invalid Credential format in Authorization header")
	}

	accessKey := strings.Split(credentialParts[1], "/")[0]
	if accessKey == "" {
		log.Println("Access Key is missing in Authorization header")
		return "", fmt.Errorf("access key is missing in Authorization header")
	}

	return accessKey, nil
}

// validateAWSSignature checks if the request has a valid AWS signature.
func validateAWSSignature(r *http.Request) bool {
	// Check for presigned URL (query string authentication)
	q := r.URL.Query()
	if q.Get("X-Amz-Signature") != "" {
		credential := q.Get("X-Amz-Credential")
		signedHeaders := q.Get("X-Amz-SignedHeaders")
		signature := q.Get("X-Amz-Signature")
		amzDate := q.Get("X-Amz-Date")
		expires := q.Get("X-Amz-Expires")

		if credential != "" && signedHeaders != "" && amzDate != "" && expires != "" {
			return aws.ValidatePresignedURLSignature(r, credential, signedHeaders, signature, amzDate)
		}
	}

	// Check for standard Authorization header signing
	authorizationHeader := r.Header.Get("Authorization")
	dateHeader := r.Header.Get("X-Amz-Date")
	amzContentSHA256 := r.Header.Get("X-Amz-Content-SHA256")

	return aws.ValidateSignature(r, authorizationHeader, dateHeader, amzContentSHA256)
}

// RetrieveBucket retrieves the bucket from the request context.
func RetrieveBucket(r *http.Request) *types.Bucket {
	bucket, ok := r.Context().Value(BucketContextKey).(*types.Bucket)
	if !ok {
		log.Println("Bucket not found in context")
		return nil
	}
	return bucket
}

// RetrieveMetadata retrieves the metadata from the request context.
func RetrieveMetadata(r *http.Request) *types.ObjectMetadata {
	metadata, ok := r.Context().Value(MetadataContextKey).(*types.ObjectMetadata)
	if !ok {
		log.Println("Metadata not found in context")
		return nil
	}
	return metadata
}

// RetrieveGrant retrieves the grant associated with the current session from the request context.
func RetrieveGrant(r *http.Request) *types.Grant {
	bucket := RetrieveBucket(r)
	if bucket == nil {
		log.Println("No bucket found in context")
		return nil
	}

	session := RetrieveSession(r)
	if session == nil {
		log.Println("No session found in context")
		return nil
	}

	for _, grant := range bucket.Grants {
		if grant.Grantee.ID == session.KeyID {
			return &grant
		}
	}

	log.Println("No matching grant found for session:", session.KeyID)
	return nil
}

// RetrieveSession retrieves the session from the request context.
func RetrieveSession(r *http.Request) *types.Authorization {
	session, ok := r.Context().Value(SessionContextKey).(*types.Authorization)
	if !ok {
		log.Println("Session state not found in context")
		return nil
	}
	return session
}

// isReadRoute checks if the request method is a read operation (GET or HEAD).
func isReadRoute(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

// isWriteRoute checks if the request method is a write operation (PUT, POST, DELETE).
func isWriteRoute(r *http.Request) bool {
	switch r.Method {
	case http.MethodPut, http.MethodPost, http.MethodDelete:
		return true
	default:
		return false
	}
}
