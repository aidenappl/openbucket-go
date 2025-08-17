package routers

import (
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
	if _, ok := q["partNumber"]; ok {
		log.Println("Currently do not support multipart uploads")
		responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", "Multipart uploads are not supported", requestId, hostId)
		return
	}
	if _, ok := q["uploadId"]; ok {
		log.Println("Currently do not support multipart uploads")
		responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", "Multipart uploads are not supported", requestId, hostId)
		return
	}
	if _, ok := q["acl"]; ok {
		log.Println("Currently do not support acl")
		responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", "ACL is not supported", requestId, hostId)
		return
	}
	if _, ok := q["tagging"]; ok {
		log.Println("Currently do not support tagging")
		responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", "Tagging is not supported", requestId, hostId)
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

func handleUpload(w http.ResponseWriter, r *http.Request) {
	// Get request variables
	vars := mux.Vars(r)
	bucket := vars["bucket"]
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
	if bucket == "" || key == "" {
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
	filePath := filepath.Join("buckets", bucket, key)

	// Validate bucket exists
	if !util.BucketExists(bucket) {
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
		w.WriteHeader(http.StatusOK)
		log.Println("Directory created:", filePath)
		return
	}

	// Create File
	eTag, err := objects.CreateObject(filePath, key, bucket, r.Body, nil, &types.UserObject{
		ID:          user.KeyID,
		DisplayName: user.Name,
	})
	if err != nil {
		http.Error(w, "Failed to create object", http.StatusInternalServerError)
		log.Println("Error creating object:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Println("File uploaded successfully. ETag:", *eTag)
}
