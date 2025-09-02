package routers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

func HandleHeadObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	objectName := vars["key"]

	// Get request and host
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)

	// Validate bucketName and objectName
	if bucketName == "" || objectName == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
			"Bucket and key must be provided", requestId, hostId)
		return
	}

	// Check if bucket exists in FS
	if !util.BucketExists(bucketName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket",
			"The specified bucket does not exist", requestId, hostId)
		return
	}

	// Check bucket in the DB
	b, err := bucket.GetBucket(bucketName)
	if err != nil || b == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", requestId, hostId)
		return
	}

	// Check if object exists
	if !objects.ObjectExists(bucketName, objectName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey",
			"Object not found", requestId, hostId)
		return
	}

	// Build object path
	objPath := filepath.Join("buckets", bucketName, objectName)

	// Get object info
	info, err := os.Stat(objPath)
	if err != nil {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		return
	}

	// If object is directory
	if info.IsDir() {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get object metadata
	meta, err := objects.GetObject(bucketName, objectName, nil)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error retrieving metadata for object", requestId, hostId)
		log.Println("Error retrieving metadata for object:", objPath)
		return
	}
	if meta == nil {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		log.Println("Error retrieving metadata for object:", objPath)
		return
	}

	// Convert to contentType
	cType := tools.ContentType(objPath)

	// Set response headers
	w.Header().Set("Content-Type", cType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	if meta.ETag != nil {
		w.Header().Set("ETag", meta.ETag.ToString())
	}

	w.WriteHeader(http.StatusOK)
}
