package routers

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
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
	p, err := auth.LoadBucketPermissions(bucket)
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to load bucket permissions", "", "")
		log.Println("Error loading bucket permissions:", err)
		return
	}

	if p == nil {
		responder.SendXML(w, http.StatusNotFound, "NoSuchBucket", "Bucket does not exist", "", "")
		log.Println("Bucket not found:", bucket)
		return
	}

	// Convert to AccessControlPolicy
	policy := &types.AccessControlPolicy{
		XmlnsXsi: types.XsiNS,
		Owner: types.UserObject{
			ID:          p.Owner.ID,
			DisplayName: p.Owner.DisplayName,
		},
		AccessControlList: p.Grants,
	}

	// Loop through grants and add types
	for i := range policy.AccessControlList {
		policy.AccessControlList[i].XmlnsXsi = types.XsiNS
		policy.AccessControlList[i].Grantee.Type = "CanonicalUser"
	}

	fmt.Println(policy)

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	if err := xml.NewEncoder(w).Encode(policy); err != nil {
		log.Println("XML encode error:", err)
	}

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
