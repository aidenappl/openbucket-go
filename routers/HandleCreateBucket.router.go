package routers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleCreateBucket(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// retrieve user grant from the request context for existing buckets
	grant := middleware.RetrieveGrant(r)

	// retrieve user session from the request context
	session := middleware.RetrieveSession(r)

	// Check if using sub query
	q := r.URL.Query()
	if _, ok := q["acl"]; ok {
		log.Println("ACL query parameter is not supported for bucket operations")
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "ACL query parameter is not supported", "", "")
		return
	}
	if _, ok := q["policy"]; ok {
		log.Println("Policy query parameter is not supported for bucket operations")
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Policy query parameter is not supported", "", "")
		return
	}
	if _, ok := q["tagging"]; ok {
		log.Println("Tagging query parameter is not supported for bucket operations")
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Tagging query parameter is not supported", "", "")
		return
	}
	if _, ok := q["versioning"]; ok {
		log.Println("Versioning query parameter is not supported for bucket operations")
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Versioning query parameter is not supported", "", "")
		return
	}
	if _, ok := q["publicAccessBlock"]; ok {
		log.Println("PublicAccessBlock query parameter is not supported for bucket operations")
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "PublicAccessBlock query parameter is not supported", "", "")
		return
	}

	// Check if any ACL headers are present
	aclHeaders := []string{
		"x-amz-grant-full-control",
		"x-amz-grant-read",
		"x-amz-grant-write",
		"x-amz-grant-read-acp",
		"x-amz-grant-write-acp",
		"x-amz-acl",
	}

	var found bool
	for _, name := range aclHeaders {
		if v := r.Header.Get(name); v != "" {
			found = true
			if name == "x-amz-acl" {
				if acl := types.ConvertToBucketACL(v); acl != types.ACLUnknown {
					log.Println("Received bucket ACL header:", name, "with value:", v)
					if !types.IsBucketACL(acl) {
						http.Error(w, fmt.Sprintf("Invalid bucket ACL value: %s", v), http.StatusBadRequest)
						return
					}
					// Handle bucket ACL
					return
				} else {
					http.Error(w, fmt.Sprintf("Invalid ACL value: %s", v), http.StatusBadRequest)
					return
				}
			}
			err := handleGrant(name, v, bucket, grant)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Handled ACL header %s for bucket %s", name, bucket)
			break
		}
	}
	if found {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if bucket == "" {
		http.Error(w, "Bucket name must be provided", http.StatusBadRequest)
		return
	}

	if err := handler.CreateBucket(bucket, types.UserObject{
		ID:          session.KeyID,
		DisplayName: session.Name,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create new grant
	newGrant := auth.NewGrant(session.KeyID, session.Name, types.FULL_CONTROL)
	if err := auth.SaveNewGrant(bucket, &newGrant); err != nil {
		log.Println("Error creating new user permissions:", err)
		http.Error(w, fmt.Sprintf("error creating new user permissions: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Bucket created successfully"))
}

func handleGrant(name string, value string, bucket string, grant *types.Grant) error {
	// Convert ACL
	reqACL := types.AWSHeaderToACL(name)
	if reqACL == types.ACLUnknown {
		log.Println("Unknown ACL header:", name)
		return fmt.Errorf("unknown ACL header: %s", name)
	}
	// Validate session has minimum permissions to handle ACL
	if grant != nil && types.IsACLModification(grant.Permission) {
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
			return fmt.Errorf("no valid ID found in ACL header value: %s", value)
		}
		// Lookup id and validate permissions
		authorization, err := auth.CheckUserExists(id)
		if err != nil {
			log.Println("Error checking user existence:", err)
			return fmt.Errorf("error checking user existence: %v", err)
		}
		if authorization == nil {
			log.Println("User with ID", id, "not found")
			return fmt.Errorf("user with ID %s not found", id)
		}

		// Check if user has existing bucket permissions
		destinationGrant, err := auth.CheckUserPermissions(id, bucket)
		if err != nil {
			log.Println("Error checking user permissions:", err)
			return fmt.Errorf("error checking user permissions: %v", err)
		}

		if destinationGrant != nil {
			// Check if requested grant is higher than existing permissions
			if destinationGrant.Permission == reqACL {
				log.Println("User already has the requested permissions for bucket:", bucket)
				return nil
			}

			// Update existing permissions
			destinationGrant.Permission = reqACL
			if err := auth.UpdateGrant(bucket, destinationGrant); err != nil {
				log.Println("Error updating user permissions:", err)
				return fmt.Errorf("error updating user permissions: %v", err)
			}
		} else {
			// Create new grant
			newGrant := auth.NewGrant(id, authorization.Name, reqACL)
			if err := auth.SaveNewGrant(bucket, &newGrant); err != nil {
				log.Println("Error creating new user permissions:", err)
				return fmt.Errorf("error creating new user permissions: %v", err)
			}
			return nil
		}

		return nil
	} else {
		log.Println("User does not have permission to modify ACLs")
		return fmt.Errorf("user does not have permission to modify ACLs")
	}
}
