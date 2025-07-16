package middleware

import (
	"context"
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
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
			keyID := r.Header.Get("X-Amz-Key-ID")
			if keyID == "" {
				responder.SendAccessDeniedXML(w, nil, nil)
				log.Println("Unauthorized: Missing KEY_ID")
				return
			}

			// Here you would check whether the user has permission to access the bucket/key
			authorized, err := auth.CheckUserPermissions(keyID, bucket)
			if err != nil || !authorized {
				responder.SendXML(w, http.StatusForbidden, "Forbidden", "You do not have permission", "", "")
				log.Printf("Forbidden: User %s does not have permission to access bucket %s", keyID, bucket)
				return
			} else {
				ctx = context.WithValue(ctx, SessionContextKey, true)
			}
		}

		// Store the context with both permissions and metadata
		r = r.WithContext(ctx)

		// Call the original handler
		handler(w, r)
	}
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

// IsSessionActive checks if the session is active based on the request context.
func IsSessionActive(r *http.Request) bool {
	session, ok := r.Context().Value(SessionContextKey).(bool)
	if !ok {
		log.Println("Session state not found in context")
		return false
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
