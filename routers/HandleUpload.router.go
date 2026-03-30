package routers

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

func HandleUpload(w http.ResponseWriter, r *http.Request) {

	// Get requestID and hostID
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)

	// Check different subrequest types
	found, _, _ := tools.HeaderExists(r, "x-amz-copy-source")
	if found {
		handler.HandleCopyObject(w, r)
		return
	}
	q := r.URL.Query()
	_, pnOk := q["partNumber"]
	_, upOk := q["uploadId"]
	if pnOk && upOk {
		handleMultipartUpload(w, r)
		return
	} else if pnOk || upOk {
		log.Println("Incomplete multipart upload")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Incomplete multipart upload", requestId, hostId)
		return
	}
	if _, ok := q["acl"]; ok {
		handlePutObjectACL(w, r)
		return
	}
	if _, ok := q["tagging"]; ok {
		handlePutObjectTagging(w, r)
		return
	}
	if _, ok := q["legal-hold"]; ok {
		log.Println("LegalHold query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "LegalHold query parameter is not supported", requestId, hostId)
		return
	}
	if _, ok := q["retention"]; ok {
		log.Println("Retention query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Retention query parameter is not supported", requestId, hostId)
		return
	}

	handleUpload(w, r)
}

func handlePutObjectTagging(w http.ResponseWriter, r *http.Request) {
	// Get request variables
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	rawKey := vars["key"]

	// Get hostID and requestID
	hostId := middleware.GetHostID(r)
	requestId := middleware.GetRequestID(r)

	// Decode key if URL-encoded (supports spaces, etc.)
	key, err := url.PathUnescape(rawKey)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidKey", "Failed to decode key", requestId, hostId)
		log.Println("Error decoding key:", err)
		return
	}

	// Validate bucket and key
	if bucketName == "" || key == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucketOrKey", "Bucket and key must be provided", requestId, hostId)
		log.Println("Bucket or key is empty")
		return
	}

	// Lookup object
	metadata := middleware.RetrieveMetadata(r)
	if metadata == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist", requestId, hostId)
		log.Println("Object not found:", key)
		return
	}

	// Decode XML body
	var taggingReq types.PutObjectTaggingRequest
	if err := xml.NewDecoder(r.Body).Decode(&taggingReq); err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "MalformedXML", "The XML you provided was not well-formed or did not validate against our published schema", requestId, hostId)
		log.Println("Failed to parse tagging XML:", err)
		return
	}

	// Delete all existing tags for overwriting
	err = objects.DeleteAllObjectTags(metadata.BucketID, metadata.ID)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Failed to delete existing object tags", requestId, hostId)
		log.Println("Error deleting existing object tags:", err)
		return
	}

	// Add new tags
	err = objects.BulkCreateObjectTags(metadata.BucketID, metadata.ID, taggingReq.TagSet)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Failed to add new object tags", requestId, hostId)
		log.Println("Error adding new object tags:", err)
		return
	}
}

func handleMultipartUpload(w http.ResponseWriter, r *http.Request) {
	// Get requestID and hostID
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)

	log.Println("Multipart upload not supported yet")
	responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", "Multipart upload not supported yet", requestId, hostId)
}

func handlePutObjectACL(w http.ResponseWriter, r *http.Request) {
	// Get request variables
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	rawKey := vars["key"]

	// Get hostID and requestID
	hostId := middleware.GetHostID(r)
	requestId := middleware.GetRequestID(r)

	// Decode key if URL-encoded (supports spaces, etc.)
	key, err := url.PathUnescape(rawKey)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidKey", "Failed to decode key", requestId, hostId)
		log.Println("Error decoding key:", err)
		return
	}

	// Validate bucket and key
	if bucketName == "" || key == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucketOrKey", "Bucket and key must be provided", requestId, hostId)
		log.Println("Bucket or key is empty")
		return
	}

	// Gather the user session
	user := middleware.RetrieveSession(r)
	if user == nil {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		log.Println("Unauthorized access attempt")
		return
	}

	// Get ACL
	acl := r.Header.Get("X-Amz-Acl")

	// Supported ACLs
	supportedACLs := map[string]bool{
		"private":                   true,
		"public-read":               true,
		"public-read-write":         false,
		"authenticated-read":        false,
		"bucket-owner-read":         false,
		"bucket-owner-full-control": false,
	}

	// Check if acl exists & is valid
	if acl == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidArgument", "ACL header is missing", requestId, hostId)
		log.Println("ACL header is missing")
		return
	}

	if valid, exists := supportedACLs[acl]; !exists || !valid {
		responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", fmt.Sprintf("ACL '%s' is not supported", acl), requestId, hostId)
		log.Println("Unsupported ACL:", acl)
		return
	}

	// Lookup object
	metadata := middleware.RetrieveMetadata(r)
	if metadata == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist", requestId, hostId)
		log.Println("Object not found:", key)
		return
	}

	// Check if update needed
	if metadata.Public && acl == "public-read" {
		w.WriteHeader(http.StatusOK)
		log.Println("ACL is already set to public-read for object:", key)
		return
	}
	if !metadata.Public && acl == "private" {
		w.WriteHeader(http.StatusOK)
		log.Println("ACL is already set to private for object:", key)
		return
	}

	// Update metadata
	if acl == "public-read" {
		metadata.Public = true
	} else {
		metadata.Public = false
	}

	// Run db update
	err = objects.UpdateObject(metadata.BucketID, metadata.Key, objects.UpdateObjectRequest{
		Public: &metadata.Public,
	})
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Failed to update object ACL", requestId, hostId)
		log.Println("Error updating object ACL:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Println("Successfully updated ACL for object:", key)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	// Get request variables
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	rawKey := vars["key"]

	// Get hostID and requestID
	hostId := middleware.GetHostID(r)
	requestId := middleware.GetRequestID(r)

	// Decode key if URL-encoded (supports spaces, etc.)
	key, err := url.PathUnescape(rawKey)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidKey", "Failed to decode key", requestId, hostId)
		log.Println("Error decoding key:", err)
		return
	}

	// Validate bucket and key
	if bucketName == "" || key == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucketOrKey", "Bucket and key must be provided", requestId, hostId)
		log.Println("Bucket or key is empty")
		return
	}

	// Gather the user session
	user := middleware.RetrieveSession(r)
	if user == nil {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		log.Println("Unauthorized access attempt")
		return
	}

	// Setup the filepath
	filePath := filepath.Join("buckets", bucketName, key)

	// Validate bucket exists
	if !util.BucketExists(bucketName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "the requested bucket does not exist", requestId, hostId)
		log.Println("requested bucket does not exist")
		return
	}

	// Get the bucket from context (loaded by middleware)
	bucketObj := middleware.RetrieveBucket(r)
	if bucketObj == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "the requested bucket does not exist", requestId, hostId)
		log.Println("requested bucket does not exist")
		return
	}

	// Check if the bucket is a directory
	isDirectory := strings.HasSuffix(key, "/")
	if isDirectory {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Failed to create directory", requestId, hostId)
			log.Println("Error creating directory:", err)
			return
		}
		// create zero byte object in db
		err = objects.CreateFolder(filePath, key, *bucketObj, *user)
		if err != nil {
			responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Failed to create directory object", requestId, hostId)
			log.Println("Error creating directory object:", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		log.Println("Directory created:", filePath)
		return
	}

	// Get ACL
	acl := r.Header.Get("X-Amz-Acl")
	public := false

	if acl != "" {
		// Supported ACLs
		supportedACLs := map[string]bool{
			"private":                   true,
			"public-read":               true,
			"public-read-write":         false,
			"authenticated-read":        false,
			"bucket-owner-read":         false,
			"bucket-owner-full-control": false,
		}

		if valid, exists := supportedACLs[acl]; !exists || !valid {
			responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", fmt.Sprintf("ACL '%s' is not supported", acl), requestId, hostId)
			log.Println("Unsupported ACL:", acl)
			return
		}

		if acl == "public-read" {
			public = true
		}
	}

	// Create File
	eTag, err := objects.CreateObject(filePath, key, *bucketObj, r.Body, nil, user, &public)
	if err != nil {
		http.Error(w, "Failed to create object", http.StatusInternalServerError)
		log.Println("Error creating object:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Println("File uploaded successfully. ETag:", *eTag)
}
