package routers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
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
			handleGrant(name, v, bucket, grant)
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

func handleGrant(name string, value string, bucket string, grant *types.Grant) {
	// Convert ACL
	reqACL := types.AWSHeaderToACL(name)
	if reqACL == types.ACLUnknown {
		log.Println("Unknown ACL header:", name)
		return
	}
	// Validate session has minimum permissions to handle ACL
	if grant != nil && types.IsACLModification(grant.ACL) {
		log.Println("Handling ACL header:", name, "with value:", value)
		var id string
		splitValue := strings.Split(value, ",")
		for _, v := range splitValue {
			if strings.HasPrefix(v, "id=") {
				sid := strings.TrimPrefix(v, "id=")
				id = strings.Trim(sid, "\"")
			}
		}
		if id == "" {
			log.Println("No valid ID found in ACL header value:", value)
			return
		}
		// Lookup id and validate permissions
		authorization, err := auth.CheckUserExists(id)
		if err != nil {
			log.Println("Error checking user existence:", err)
			return
		}
		if authorization == nil {
			log.Println("User with ID", id, "not found")
			return
		}

		// Check if user has existing bucket permissions
		userGrant, err := auth.CheckUserPermissions(id, bucket)
		if err != nil {
			log.Println("Error checking user permissions:", err)
			return
		}

		if userGrant != nil {
			// Check if requested grant is higher than existing permissions
			if userGrant.ACL == reqACL {
				log.Println("User already has the requested permissions for bucket:", bucket)
				return
			}
		} else {
			fmt.Println("User does not have existing permissions for bucket:", bucket)
			return
		}

		return
	} else {
		log.Println("User does not have permission to modify ACLs")
		return
	}
}
