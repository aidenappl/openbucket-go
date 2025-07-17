package routers

import (
	"log"
	"net/http"
	"strings"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleCreateBucket(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// retrieve user grant from the request context
	grant := middleware.RetrieveGrant(r)

	// Check if any ACL headers are present
	aclHeaders := []string{
		"x-amz-grant-full-control",
		"x-amz-grant-read",
		"x-amz-grant-write",
		"x-amz-grant-read-acp",
		"x-amz-grant-write-acp",
	}

	var found bool
	for _, name := range aclHeaders {
		if v := r.Header.Get(name); v != "" {
			found = true
			handleGrant(name, v, grant)
		}
	}

	if found {
		http.Error(w, "ACL headers are not supported", http.StatusBadRequest)
		return
	}

	if bucket == "" {
		http.Error(w, "Bucket name must be provided", http.StatusBadRequest)
		return
	}

	if err := handler.CreateBucket(bucket); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Bucket created successfully"))
}

func handleGrant(name, value string, grant *types.Grant) {
	// Validate session has minimum permissions to handle ACL
	if grant != nil && types.IsACLModification(grant.ACL) {
		log.Println("Handling ACL header:", name, "with value:", value)
		var id string
		splitValue := strings.Split(value, ",")
		for _, v := range splitValue {
			if strings.HasPrefix(v, "id=") {
				id = strings.TrimPrefix(v, "id=")
			}
		}
		if id == "" {
			log.Println("No valid ID found in ACL header value:", value)
			return
		}
		// Lookup id and validate permissions
		
		return
	} else {
		log.Println("User does not have permission to modify ACLs")
		return
	}
}
