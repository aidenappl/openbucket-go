package routers

import (
	"net/http"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

// HandleBucketHead handles probes for bucket-exists checks & permissions checks
func HandleBucketHead(w http.ResponseWriter, r *http.Request) {
	// Get mux variables
	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	// Get request & host
	requestID := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Check if bucket exists
	if !util.BucketExists(bucketName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", requestID, host)
		return
	}

	// Check if bucket exists in db
	b, err := bucket.GetBucket(bucketName)
	if err != nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", requestID, host)
		return
	}

	if b == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist", requestID, host)
		return
	}

	// Get session from context
	session := middleware.RetrieveSession(r)
	if session == nil {
		responder.SendAccessDeniedXML(w, &requestID, &host)
		return
	}

	// Check if user has permission to bucket
	grant, err := auth.CheckUserPermissions(session.KeyID, bucketName)
	if err != nil || grant == nil {
		responder.SendAccessDeniedXML(w, &requestID, &host)
		return
	}

	w.Header().Set("X-Amz-Meta-Owner-Id", b.Owner.ID)
	w.Header().Set("X-Amz-Meta-Owner-Display-Name", b.Owner.DisplayName)
	w.Header().Set("X-Amz-Acl", string(b.ACL))

	// If all checks pass, return 200 OK
	w.WriteHeader(http.StatusOK)
}
