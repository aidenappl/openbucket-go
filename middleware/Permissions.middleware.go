package middleware

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/aws"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

type contextKey string

var PermissionsContextKey contextKey = "permissions"
var MetadataContextKey contextKey = "metadata"
var SessionContextKey contextKey = "session"

// Example wrapper function that checks permissions before invoking the handler
func Authorized(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the bucket and key from the URL
		vars := mux.Vars(r)
		bucket := vars["bucket"]
		key := vars["key"]

		// Initialize context with the original request context
		ctx := r.Context()

		// Load the bucket's permissions
		var permissions *types.Permissions
		var metadata *types.Metadata

		// Check AWS Signature (if valid)
		if !validateAWSSignature(r) {
			responder.SendAccessDeniedXML(w, nil, nil)
			return
		}

		// Load bucket permissions
		if bucket != "" {
			perms, err := auth.LoadPermissions(bucket)
			if err != nil {
				responder.SendAccessDeniedXML(w, nil, nil)
				log.Println("Error loading permissions for bucket:", bucket, err)
				return
			}

			permissions = perms
			// Store the permissions in the context
			ctx = context.WithValue(ctx, PermissionsContextKey, permissions)
		}

		// Load metadata if the key is not empty
		if key != "" {
			metadataFilePath := filepath.Join("buckets", bucket, key+".obmeta")
			if _, err := os.Stat(metadataFilePath); err == nil {
				// If .obmeta exists, read and parse the metadata XML
				metadataFile, err := os.Open(metadataFilePath)
				if err != nil {
					responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to open .obmeta file", "", "")
					log.Println("Error opening .obmeta file:", err)
					return
				}
				defer metadataFile.Close()

				// Parse the XML metadata
				decoder := xml.NewDecoder(metadataFile)
				if err := decoder.Decode(&metadata); err != nil {
					responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Error parsing metadata XML", "", "")
					log.Println("Error parsing .obmeta XML:", err)
					return
				}

				// Store the metadata in the context
				ctx = context.WithValue(ctx, MetadataContextKey, metadata)
			}
		}

		// Validate key is not obmeta (skip checking for metadata files)
		if strings.HasSuffix(key, ".obmeta") {
			responder.SendAccessDeniedXML(w, nil, nil)
			log.Println("Attempted to access metadata file directly:", key)
			return
		}

		// Check if permissions are needed
		if permissions != nil && permissions.AllowGlobalWrite && isWriteRoute(r) ||
			permissions != nil && permissions.AllowGlobalRead && isReadRoute(r) ||
			metadata != nil && metadata.Public && isReadRoute(r) {
			handler(w, r)
			return
		}

		// If permissions are required, check if the user has access
		if permissions != nil && metadata != nil && !permissions.AllowGlobalRead && !metadata.Public {
			// Check the permissions for the user (you would have your own logic for this)
			keyID, err := GetAccessKeyFromRequest(r)
			if err != nil {
				responder.SendXML(w, http.StatusUnauthorized, "Unauthorized", "Missing or invalid access key", "", "")
				log.Println("Unauthorized: Missing or invalid access key")
				return
			}

			// Here you would check whether the user has permission to access the bucket/key
			authorized, err := auth.CheckUserPermissions(keyID, bucket)
			if err != nil || authorized == nil {
				responder.SendXML(w, http.StatusForbidden, "Forbidden", "You do not have permission", "", "")
				log.Printf("Forbidden: User %s does not have permission to access bucket %s", keyID, bucket)
				return
			} else {
				ctx = context.WithValue(ctx, SessionContextKey, authorized)
			}
		}

		// Store the context with both permissions and metadata
		r = r.WithContext(ctx)

		// Call the original handler
		handler(w, r)
	}
}

func GetAccessKeyFromRequest(r *http.Request) (string, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	// Extract the AWS access key and signature from the Authorization header
	parts := strings.Split(authorizationHeader, " ")
	if parts[0] != "AWS4-HMAC-SHA256" {
		log.Println("Invalid Authorization header format")
		return "", fmt.Errorf("invalid Authorization header format")
	}

	// Extract the credential part from the Authorization header
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

// Validate AWS Signature
func validateAWSSignature(r *http.Request) bool {
	// Extract necessary headers for validation
	authorizationHeader := r.Header.Get("Authorization")
	dateHeader := r.Header.Get("X-Amz-Date")
	amzContentSHA256 := r.Header.Get("X-Amz-Content-SHA256")

	// You can then pass these headers along with the request and verify them
	// using AWS Signature Version 4 signing process or your own logic
	return aws.ValidateSignature(r, authorizationHeader, dateHeader, amzContentSHA256)
}

// RetrievePermissions retrieves the permissions from the context.
func RetrievePermissions(r *http.Request) *types.Permissions {
	permissions, ok := r.Context().Value(PermissionsContextKey).(*types.Permissions)
	if !ok {
		log.Println("Permissions not found in context")
		return nil
	}
	return permissions
}

// RetrieveMetadata retrieves the metadata from the context.
func RetrieveMetadata(r *http.Request) *types.Metadata {
	metadata, ok := r.Context().Value(MetadataContextKey).(*types.Metadata)
	if !ok {
		log.Println("Metadata not found in context")
		return nil
	}
	return metadata
}

// RetrieveSession retrieves the session information from the context.
func RetrieveSession(r *http.Request) *types.Authorization {
	session, ok := r.Context().Value(SessionContextKey).(*types.Authorization)
	if !ok {
		log.Println("Session state not found in context")
		return nil
	}
	return session
}

func isReadRoute(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

func isWriteRoute(r *http.Request) bool {
	switch r.Method {
	case http.MethodPut, http.MethodPost, http.MethodDelete:
		return true
	default:
		return false
	}
}
