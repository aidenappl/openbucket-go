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
	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

type contextKey string

var PermissionsContextKey contextKey = "permissions"
var MetadataContextKey contextKey = "metadata"
var SessionContextKey contextKey = "session"

func Authorized(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		bucket := vars["bucket"]
		key := vars["key"]

		ctx := r.Context()

		var permissions *types.Permissions
		var metadata *types.Metadata

		// Get request and host id from context
		requestID := GetRequestID(r)
		hostID := GetHostID(r)

		if bucket != "" {
			perms, err := auth.LoadPermissions(bucket)
			if err != nil {
				responder.SendAccessDeniedXML(w, &requestID, &hostID)
				log.Println("Error loading permissions for bucket:", bucket, err)
				return
			}

			permissions = perms

			ctx = context.WithValue(ctx, PermissionsContextKey, permissions)
		}

		if validateAWSSignature(r) {
			log.Println("Valid AWS signature for request:", r.Method, r.URL.Path)
		} else {
			log.Println("Invalid AWS signature for request:", r.Method, r.URL.Path)
			responder.SendAccessDeniedXML(w, &requestID, &hostID)
			return
		}

		if key != "" {
			metadataFilePath := filepath.Join("buckets", bucket, key+".obmeta")
			if _, err := os.Stat(metadataFilePath); err == nil {

				metadataFile, err := os.Open(metadataFilePath)
				if err != nil {
					responder.SendAccessDeniedXML(w, &requestID, &hostID)
					log.Println("Error opening .obmeta file:", err)
					return
				}
				defer metadataFile.Close()

				decoder := xml.NewDecoder(metadataFile)
				if err := decoder.Decode(&metadata); err != nil {
					responder.SendAccessDeniedXML(w, &requestID, &hostID)
					log.Println("Error parsing .obmeta XML:", err)
					return
				}

				ctx = context.WithValue(ctx, MetadataContextKey, metadata)
			}
		}

		if strings.HasSuffix(key, ".obmeta") {
			responder.SendAccessDeniedXML(w, &requestID, &hostID)
			log.Println("Attempted to access metadata file directly:", key)
			return
		}

		if permissions != nil && permissions.AllowGlobalWrite && isWriteRoute(r) ||
			permissions != nil && permissions.AllowGlobalRead && isReadRoute(r) ||
			metadata != nil && metadata.Public && isReadRoute(r) {
			log.Println("Bypassing permissions check due to global or public access")
			r = r.WithContext(ctx)
			handler(w, r)
			return
		}

		keyID, err := GetAccessKeyFromRequest(r)
		if err != nil {
			responder.SendAccessDeniedXML(w, &requestID, &hostID)
			log.Println("Unauthorized: Missing or invalid access key")
			return
		}

		// Validate signature if accessing within bucket
		if bucket != "" {
			authorized, err := auth.CheckUserPermissions(keyID, bucket)
			if err != nil {
				responder.SendAccessDeniedXML(w, &requestID, &hostID)
				log.Println("Error checking user permissions:", err)
				return
			}
			if authorized != nil {
				if isWriteRoute(r) && types.IsWritePermission(authorized.ACL) {
					responder.SendAccessDeniedXML(w, &requestID, &hostID)
					log.Printf("Forbidden: User %s does not have write permission for bucket %s", keyID, bucket)
					return
				}
				if isReadRoute(r) && !types.IsReadPermission(authorized.ACL) {
					responder.SendAccessDeniedXML(w, &requestID, &hostID)
					log.Printf("Forbidden: User %s does not have read permission for bucket %s", keyID, bucket)
					return
				}
				ctx = context.WithValue(ctx, SessionContextKey, authorized)
			} else {
				responder.SendAccessDeniedXML(w, &requestID, &hostID)
				log.Printf("Forbidden: User %s does not have permission to access bucket %s", keyID, bucket)
				return
			}
		} else {
			authorized, err := auth.CheckUserExists(keyID)
			if err != nil {
				responder.SendAccessDeniedXML(w, &requestID, &hostID)
				log.Println("Error checking user permissions:", err)
				return
			}
			if authorized != nil {
				ctx = context.WithValue(ctx, SessionContextKey, authorized)
			} else {
				responder.SendAccessDeniedXML(w, &requestID, &hostID)
				log.Printf("Forbidden: User %s does not have permission to access bucket %s", keyID, bucket)
				return
			}
		}

		r = r.WithContext(ctx)
		handler(w, r)
	}
}

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

func validateAWSSignature(r *http.Request) bool {

	authorizationHeader := r.Header.Get("Authorization")
	dateHeader := r.Header.Get("X-Amz-Date")
	amzContentSHA256 := r.Header.Get("X-Amz-Content-SHA256")

	return aws.ValidateSignature(r, authorizationHeader, dateHeader, amzContentSHA256)
}

func RetrievePermissions(r *http.Request) *types.Permissions {
	permissions, ok := r.Context().Value(PermissionsContextKey).(*types.Permissions)
	if !ok {
		log.Println("Permissions not found in context")
		return nil
	}
	return permissions
}

func RetrieveMetadata(r *http.Request) *types.Metadata {
	metadata, ok := r.Context().Value(MetadataContextKey).(*types.Metadata)
	if !ok {
		log.Println("Metadata not found in context")
		return nil
	}
	return metadata
}

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
