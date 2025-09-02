package routers

import (
	"encoding/xml"
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

func HandleBucket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	if bucket == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Bucket name is required", "", "")
		return
	}

	q := r.URL.Query()
	if _, ok := q["acl"]; ok {
		HandleBucketACL(w, r, bucket)
		return
	}
	if _, ok := q["session"]; ok {
		log.Println("Session query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Session query parameter is not supported", "", "")
		return
	}
	if _, ok := q["location"]; ok {
		handleGetBucketLocation(w, r, bucket)
		return
	}
	if _, ok := q["policy"]; ok {
		handleGetBucketPolicy(w, r, bucket)
		return
	}
	if _, ok := q["tagging"]; ok {
		handleGetBucketTagging(w, r, bucket)
		return
	}
	if _, ok := q["versioning"]; ok {
		log.Println("Versioning query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Versioning query parameter is not supported", "", "")
		return
	}
	if _, ok := q["publicAccessBlock"]; ok {
		log.Println("PublicAccessBlock query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "PublicAccessBlock query parameter is not supported", "", "")
		return
	}
	if _, ok := q["uploads"]; ok {
		log.Println("Uploads query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Uploads query parameter is not supported", "", "")
		return
	}
	if _, ok := q["versions"]; ok {
		log.Println("Versions query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Versions query parameter is not supported", "", "")
		return
	}
	if _, ok := q["accelerate"]; ok {
		log.Println("Accelerate query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Accelerate query parameter is not supported", "", "")
		return
	}
	if _, ok := q["analytics"]; ok {
		log.Println("Analytics query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Analytics query parameter is not supported", "", "")
		return
	}
	if _, ok := q["cors"]; ok {
		log.Println("CORS query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "CORS query parameter is not supported", "", "")
		return
	}
	if _, ok := q["encryption"]; ok {
		log.Println("Encryption query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Encryption query parameter is not supported", "", "")
		return
	}
	if _, ok := q["intelligent-tiering"]; ok {
		log.Println("Intelligent-Tiering query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Intelligent-Tiering query parameter is not supported", "", "")
		return
	}
	if _, ok := q["inventory"]; ok {
		log.Println("Inventory query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Inventory query parameter is not supported", "", "")
		return
	}
	if _, ok := q["lifecycle"]; ok {
		log.Println("Lifecycle query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Lifecycle query parameter is not supported", "", "")
		return
	}
	if _, ok := q["logging"]; ok {
		log.Println("Logging query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Logging query parameter is not supported", "", "")
		return
	}
	if _, ok := q["metadataTable"]; ok {
		log.Println("MetadataTable query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "MetadataTable query parameter is not supported", "", "")
		return
	}
	if _, ok := q["metrics"]; ok {
		log.Println("Metrics query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Metrics query parameter is not supported", "", "")
		return
	}
	if _, ok := q["notification"]; ok {
		log.Println("Notification query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Notification query parameter is not supported", "", "")
		return
	}
	if _, ok := q["ownershipControls"]; ok {
		log.Println("OwnershipControls query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "OwnershipControls query parameter is not supported", "", "")
		return
	}
	if _, ok := q["policyStatus"]; ok {
		log.Println("PolicyStatus query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "PolicyStatus query parameter is not supported", "", "")
		return
	}
	if _, ok := q["replication"]; ok {
		log.Println("Replication query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Replication query parameter is not supported", "", "")
		return
	}
	if _, ok := q["requestPayment"]; ok {
		log.Println("RequestPayment query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "RequestPayment query parameter is not supported", "", "")
		return
	}
	if _, ok := q["website"]; ok {
		log.Println("RequestPayment query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "RequestPayment query parameter is not supported", "", "")
		return
	}

	HandleListObjects(w, r)
}

func handleGetBucketTagging(w http.ResponseWriter, r *http.Request, bucketName string) {
	// Get hostId and requestId
	hostID := middleware.GetHostID(r)
	reqID := middleware.GetRequestID(r)

	// Validate bucket
	if bucketName == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucket", "Bucket name is required", reqID, hostID)
		return
	}

	// Get bucket from context
	b := middleware.RetrieveBucket(r)

	// Get tags from the db
	tags, err := bucket.GetBucketTags(b.ID)
	if err != nil {
		log.Println("Error retrieving bucket tags:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Unable to retrieve bucket tags", reqID, hostID)
		return
	}

	response := types.GetBucketTaggingResponse{
		Xmlns:  types.XsiNS_Default,
		TagSet: *tags,
	}

	responder.SendXML(w, http.StatusOK, response)
}

func handleGetBucketPolicy(w http.ResponseWriter, r *http.Request, bucket string) {
	// Get hostId and requestId
	hostID := middleware.GetHostID(r)
	reqID := middleware.GetRequestID(r)

	// Validate bucket
	if bucket == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucket", "Bucket name is required", reqID, hostID)
		return
	}

	// Verify the bucket does exist
	if !util.BucketExists(bucket) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket",
			"The specified bucket does not exist", reqID, hostID)
		return
	}

	// Respond with no policy since we don't have policy management
	responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucketPolicy",
		"The bucket policy does not exist", reqID, hostID)
}

func handleGetBucketLocation(w http.ResponseWriter, r *http.Request, bucket string) {
	// Get hostId and requestId
	hostId := middleware.GetHostID(r)
	requestId := middleware.GetRequestID(r)

	// Validate bucket
	if bucket == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucket", "Bucket name is required", requestId, hostId)
		return
	}

	// Get region
	region := env.Region

	// Build response
	var response types.GetBucketLocationResponse
	response.LocationConstraint = &region

	responder.SendXML(w, http.StatusOK, response)
}

func HandleBucketACL(w http.ResponseWriter, r *http.Request, bucketName string) {
	// Implementation for handling bucket ACL
	p, err := bucket.GetBucket(bucketName)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Unable to load bucket permissions", "", "")
		log.Println("Error loading bucket permissions:", err)
		return
	}

	if p == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "Bucket does not exist", "", "")
		log.Println("Bucket not found:", bucketName)
		return
	}

	// Convert to AccessControlPolicy
	policy := &types.AccessControlPolicyResponse{
		XmlnsXsi: types.XsiNS,
		Owner: types.UserObject{
			ID:          p.Owner.ID,
			DisplayName: p.Owner.DisplayName,
		},
		AccessControlList: p.Grants,
	}

	responder.SendXML(w, http.StatusOK, policy)
}

func HandleListObjects(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	q := r.URL.Query()

	if q.Has("acl") {
		log.Println("ACL query parameter is not supported for listing objects")
	}

	objectList, err := objects.ListObjectsXML(bucket, r.URL.Query())
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Unable to list objects", "", "")
		log.Println("Error listing objects:", err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")

	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(objectList)
}
