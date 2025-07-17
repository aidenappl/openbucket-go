package middleware

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/aws"
	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

type contextKey string

var PermissionsContextKey contextKey = "permissions"
var MetadataContextKey contextKey = "metadata"
var SessionContextKey contextKey = "session"

// Authorized is a middleware that checks if the user is authorized to access the requested resource.
func Authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucket, key := vars["bucket"], vars["key"]
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
		perms, err := loadBucketPermissions(bucket)
		if err != nil {
			deny("Error loading permissions for bucket "+bucket, err)
			return
		}
		if perms != nil {
			ctx = context.WithValue(ctx, PermissionsContextKey, perms)
		}

		// Deny access to metadata files directly
		if strings.HasSuffix(key, ".obmeta") {
			deny("Attempted to access metadata file directly: "+key, nil)
			return
		}

		// Load object metadata if available
		if md, err := loadObjectMetadata(bucket, key); err == nil {
			ctx = context.WithValue(ctx, MetadataContextKey, md)
		} else if !errors.Is(err, os.ErrNotExist) {
			deny("Error loading object metadata", err)
			return
		}

		// Do a fast path check for public access or ACL permissions
		if isFastPathAllowed(perms, ctx, r) {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
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
		if bucket == "" {
			session, err := auth.CheckUserExists(keyID)
			if err != nil {
				deny("Unauthorized: "+err.Error(), nil)
				return
			}
			ctx = context.WithValue(ctx, SessionContextKey, session)
		}

		// Authorize against the bucket ACL
		session, err := authoriseByACL(keyID, bucket, r)
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

// returns nil when bucket == "" (root routes)
func loadBucketPermissions(bucket string) (*types.Permissions, error) {
	if bucket == "" {
		return nil, nil
	}
	return auth.LoadPermissions(bucket)
}

// loadObjectMetadata loads the metadata for the specified object.
func loadObjectMetadata(bucket, key string) (*types.Metadata, error) {
	if bucket == "" || key == "" {
		return nil, nil
	}
	metaPath := filepath.Join("buckets", bucket, key+".obmeta")
	f, err := os.Open(metaPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var md types.Metadata
	if err := xml.NewDecoder(f).Decode(&md); err != nil {
		return nil, err
	}
	return &md, nil
}

// isFastPathAllowed checks if the request can be served without further permission checks
func isFastPathAllowed(perms *types.Permissions, ctx context.Context, r *http.Request) bool {
	if perms != nil {
		if isWriteRoute(r) && types.IsBucketACLWrite(perms.ACL) {
			return true
		}
		if isReadRoute(r) && types.IsBucketACLRead(perms.ACL) {
			return true
		}
	}

	// Object‑level public flag
	if mdRaw := r.Context().Value(MetadataContextKey); mdRaw != nil && isReadRoute(r) {
		if md, ok := mdRaw.(*types.Metadata); ok && md.Public {
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
	case isWriteRoute(r) && !types.IsWritePermission(userACL.ACL):
		return nil, fmt.Errorf("user %s lacks write permission on bucket %s", keyID, bucket)
	case isReadRoute(r) && !types.IsReadPermission(userACL.ACL):
		return nil, fmt.Errorf("user %s lacks read permission on bucket %s", keyID, bucket)
	}
	return session, nil
}

// GetAccessKeyFromRequest extracts the access key from the request's Authorization header.
func GetAccessKeyFromRequest(r *http.Request) (string, error) {

	if env.BypassPermissions {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			return "", fmt.Errorf("authorization header is missing")
		}
		return authorizationHeader, nil
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

	authorizationHeader := r.Header.Get("Authorization")
	dateHeader := r.Header.Get("X-Amz-Date")
	amzContentSHA256 := r.Header.Get("X-Amz-Content-SHA256")

	return aws.ValidateSignature(r, authorizationHeader, dateHeader, amzContentSHA256)
}

// RetrievePermissions retrieves the permissions from the request context.
func RetrievePermissions(r *http.Request) *types.Permissions {
	permissions, ok := r.Context().Value(PermissionsContextKey).(*types.Permissions)
	if !ok {
		log.Println("Permissions not found in context")
		return nil
	}
	return permissions
}

// RetrieveMetadata retrieves the metadata from the request context.
func RetrieveMetadata(r *http.Request) *types.Metadata {
	metadata, ok := r.Context().Value(MetadataContextKey).(*types.Metadata)
	if !ok {
		log.Println("Metadata not found in context")
		return nil
	}
	return metadata
}

// RetrieveGrant retrieves the grant associated with the current session from the request context.
func RetrieveGrant(r *http.Request) *types.Grant {
	permissions := RetrievePermissions(r)
	if permissions == nil {
		log.Println("No permissions found in context")
		return nil
	}

	session := RetrieveSession(r)
	if session == nil {
		log.Println("No session found in context")
		return nil
	}

	for _, grant := range permissions.Grants {
		if grant.KeyID == session.KeyID {
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
