package routers

import (
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

func HandleDeleteBucket(w http.ResponseWriter, r *http.Request) {
	// Query Param Redirectors
	query := r.URL.Query()
	if _, ok := query["policy"]; ok {
		log.Println("Policy query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Policy query parameter is not supported", "", "")
		return
	}
	if _, ok := query["tagging"]; ok {
		log.Println("Tagging query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Tagging query parameter is not supported", "", "")
		return
	}
	if _, ok := query["publicAccessBlock"]; ok {
		log.Println("PublicAccessBlock query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "PublicAccessBlock query parameter is not supported", "", "")
		return
	}
	if _, ok := query["analytics"]; ok {
		log.Println("Analytics query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Analytics query parameter is not supported", "", "")
		return
	}
	if _, ok := query["cors"]; ok {
		log.Println("CORS query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "CORS query parameter is not supported", "", "")
		return
	}
	if _, ok := query["encryption"]; ok {
		log.Println("Encryption query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Encryption query parameter is not supported", "", "")
		return
	}
	if _, ok := query["intelligent-tiering"]; ok {
		log.Println("Intelligent-Tiering query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Intelligent-Tiering query parameter is not supported", "", "")
		return
	}
	if _, ok := query["inventory"]; ok {
		log.Println("Inventory query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Inventory query parameter is not supported", "", "")
		return
	}
	if _, ok := query["lifecycle"]; ok {
		log.Println("Lifecycle query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Lifecycle query parameter is not supported", "", "")
		return
	}
	if _, ok := query["metadataTable"]; ok {
		log.Println("MetadataTable query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "MetadataTable query parameter is not supported", "", "")
		return
	}
	if _, ok := query["metrics"]; ok {
		log.Println("Metrics query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Metrics query parameter is not supported", "", "")
		return
	}
	if _, ok := query["ownershipControls"]; ok {
		log.Println("OwnershipControls query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "OwnershipControls query parameter is not supported", "", "")
		return
	}
	if _, ok := query["replication"]; ok {
		log.Println("Replication query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Replication query parameter is not supported", "", "")
		return
	}
	if _, ok := query["website"]; ok {
		log.Println("Website query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Website query parameter is not supported", "", "")
		return
	}

	HandleBucketDeletion(w, r)
}

func HandleBucketDeletion(w http.ResponseWriter, r *http.Request) {
	// Get the bucket from the url
	bucketName := mux.Vars(r)["bucket"]

	// Validate the bucket
	if bucketName == "" {
		log.Println("Bucket name is required")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Bucket name is required", "", "")
		return
	}

	// Check if the bucket exists
	if !util.BucketExists(bucketName) {
		log.Println("Bucket does not exist")
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", "", "")
		return
	}

	// Validate owner of bucket
	// TODO: use IAM based permissions to check for s3:DeleteBucket

	// retrieve bucket permissions from context
	bucketMetadata := middleware.RetrievePermissions(r)
	if bucketMetadata == nil || bucketMetadata.Name != bucketName {
		log.Println("Unauthorized access to bucket deletion")
		responder.SendXMLError(w, http.StatusForbidden, "AccessDenied", "You do not have permission to delete this bucket", "", "")
		return
	}

	// retrieve user session from context
	userSession := middleware.RetrieveSession(r)
	if userSession == nil {
		log.Println("Unauthorized access attempt")
		responder.SendXMLError(w, http.StatusUnauthorized, "Unauthorized", "You must be authorized to access this endpoint", "", "")
		return
	}

	// Validate that bucket owner id is the same user accessing api
	if bucketMetadata.Owner.ID != userSession.KeyID {
		log.Println("Unauthorized access to bucket deletion")
		responder.SendXMLError(w, http.StatusForbidden, "AccessDenied", "You do not have permission to delete this bucket", "", "")
		return
	}

	// Validate bucket is empty
	objects, err := handler.ListObjects(bucketName)
	if err != nil {
		log.Println("Error listing objects in bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error listing objects in bucket", "", "")
		return
	}

	// If there are objects conflict occurs
	if len(objects) > 0 {
		log.Println("Bucket is not empty, cannot delete")
		responder.SendXMLError(w, http.StatusConflict, "BucketNotEmpty", "The specified bucket is not empty", "", "")
		return
	}

	// Handle bucket deletion
	err = bucket.DeleteBucket(bucketName)
	if err != nil {
		log.Println("Error deleting bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error deleting bucket", "", "")
		return
	}

	// Success Response
	w.WriteHeader(http.StatusNoContent)
}
