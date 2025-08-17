package routers

import (
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	// Check sub queries
	q := r.URL.Query()
	if _, ok := q["uploadId"]; ok {
		log.Println("Currently do not support multipart cancellations")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}
	if _, ok := q["tagging"]; ok {
		log.Println("Currently do not support tagging")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	deleteObject(w, r)
}

func deleteObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	objectName := vars["key"]

	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)

	// validate bucket name and object name
	if bucketName == "" || objectName == "" {
		responder.SendXMLError(w, http.StatusNotFound, "InvalidBucketOrObject", "The bucket name or object name is invalid", requestId, hostId)
		return
	}

	// Check that bucket exists
	if !util.BucketExists(bucketName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", requestId, hostId)
		return
	}

	// Check that object exists
	if !util.ObjectExists(bucketName, objectName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist", requestId, hostId)
		return
	}

	// Delete the object
	err := objects.DeleteObject(bucketName, objectName)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Failed to delete object", requestId, hostId)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	log.Printf("Successfully deleted object %s from bucket %s", objectName, bucketName)
}
