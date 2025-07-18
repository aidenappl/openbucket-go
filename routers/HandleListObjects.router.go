package routers

import (
	"encoding/xml"
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/gorilla/mux"
)

func HandleBucket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	q := r.URL.Query()
	if _, ok := q["acl"]; ok {
		HandleBucketACL(w, r, bucket)
		return
	}
	HandleListObjects(w, r)
}

func HandleBucketACL(w http.ResponseWriter, r *http.Request, bucket string) {
	// Implementation for handling bucket ACL
}

func HandleListObjects(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	q := r.URL.Query()

	if q.Has("acl") {
		log.Println("ACL query parameter is not supported for listing objects")
	}

	objectList, err := handler.ListObjectsXML(bucket, r.URL.Query())
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to list objects", "", "")
		log.Println("Error listing objects:", err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")

	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(objectList)
}
