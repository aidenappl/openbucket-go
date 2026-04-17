package routers

import (
	"encoding/xml"
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleModifyBucket(w http.ResponseWriter, r *http.Request) {
	// validate sub queries
	q := r.URL.Query()
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)
	if _, ok := q["delete"]; ok {
		handleDeleteObjects(w, r)
		return
	}
	if _, ok := q["metadataTable"]; ok {
		log.Println("Metadata Table query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Metadata Table query parameter is not supported", requestId, hostId)
		return
	}
}

func handleDeleteObjects(w http.ResponseWriter, r *http.Request) {
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)
	bucketName := mux.Vars(r)["bucket"]

	// validate bucketName
	if bucketName == "" {
		log.Println("Bucket name is required")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Bucket name is required", requestId, hostId)
		return
	}

	// validate bucket exists
	b, err := bucket.GetBucket(bucketName)
	if err != nil {
		log.Println("Bucket does not exist")
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "Bucket does not exist", requestId, hostId)
		return
	}

	if b == nil {
		log.Println("Bucket does not exist")
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "Bucket does not exist", requestId, hostId)
		return
	}

	// parse body to DeleteRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var deleteRequest types.DeleteRequest
	if err := xml.NewDecoder(r.Body).Decode(&deleteRequest); err != nil {
		log.Printf("Failed to parse delete request body: %v", err)
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Failed to parse delete request body", requestId, hostId)
		return
	}

	// Validate there are objects to delete
	if len(deleteRequest.Objects) == 0 {
		log.Println("No objects specified for deletion")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "No objects specified for deletion", requestId, hostId)
		return
	}

	var deleteResponse types.DeleteResult

	// process deleteRequest
	for _, obj := range deleteRequest.Objects {
		if err := objects.DeleteVersionedObject(*b, obj.Key, obj.VersionId); err != nil {
			deleteResponse.Errors = append(deleteResponse.Errors, types.DeleteErrorEntry{
				Key:     obj.Key,
				Code:    "InternalError",
				Message: err.Error(),
			})
		} else {
			deleteResponse.Deleted = append(deleteResponse.Deleted, types.DeletedObject{
				Key:       obj.Key,
				VersionId: obj.VersionId,
			})
		}
	}

	// Send the response
	responder.SendXML(w, http.StatusOK, deleteResponse)

}
