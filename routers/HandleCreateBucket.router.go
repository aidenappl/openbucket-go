package routers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleCreateBucket(w http.ResponseWriter, r *http.Request) {

	// Check if using sub query
	q := r.URL.Query()
	if _, ok := q["acl"]; ok {
		log.Println("ACL query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "ACL query parameter is not supported", "", "")
		return
	}
	if _, ok := q["policy"]; ok {
		log.Println("Policy query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Policy query parameter is not supported", "", "")
		return
	}
	if _, ok := q["tagging"]; ok {
		log.Println("Tagging query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Tagging query parameter is not supported", "", "")
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
		log.Println("Inventory query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Inventory query parameter is not supported", "", "")
		return
	}
	if _, ok := q["logging"]; ok {
		log.Println("Logging query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Logging query parameter is not supported", "", "")
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
		log.Println("Website query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Website query parameter is not supported", "", "")
		return
	}
	if _, ok := q["object-lock"]; ok {
		log.Println("ObjectLock query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "ObjectLock query parameter is not supported", "", "")
		return
	}

	// retrieve user grant from the request context for existing buckets
	grant := middleware.RetrieveGrant(r)

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
			vars := mux.Vars(r)
			bucket := vars["bucket"]
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

	handleNewBucket(w, r)
}

func handleNewBucket(w http.ResponseWriter, r *http.Request) {
	// Get mux bucket variable
	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	// Get the request id & host id
	requestID := middleware.GetRequestID(r)
	hostID := middleware.GetHostID(r)

	// Validate bucket
	if bucketName == "" {
		responder.SendAccessDeniedXML(w, &requestID, &hostID)
		return
	}

	// Retrieve User Session
	session := middleware.RetrieveSession(r)

	// validate session
	if session == nil {
		responder.SendAccessDeniedXML(w, &requestID, &hostID)
		return
	}

	// TODO: integrate with IAM to verify permissions, right now if an authorized user anyone can create buckets

	// Get bucket, check if exists
	b, err := bucket.GetBucket(bucketName)
	if err != nil {
		log.Println("Error getting bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Error getting bucket: %v", err), requestID, hostID)
		return
	}

	if b != nil {
		log.Println("Bucket already exists:", bucketName)
		responder.SendXMLError(w, http.StatusConflict, "BucketAlreadyExists", fmt.Sprintf("Bucket already exists: %s", bucketName), requestID, hostID)
		return
	}

	// Create Base Bucket
	if err := bucket.CreateBucket(bucketName, session.ID); err != nil {
		log.Println("Error creating bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Error creating bucket: %v", err), requestID, hostID)
		return
	}

	// Success response
	responder.SendXML(w, http.StatusOK, types.CreateBucketResult{
		Location: fmt.Sprintf("/%s", bucketName),
	})
}

func handleGrant(name string, value string, bucketName string, grant *types.Grant) error {
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
		destinationGrant, err := auth.CheckUserPermissions(id, bucketName)
		if err != nil {
			log.Println("Error checking user permissions:", err)
			return fmt.Errorf("error checking user permissions: %v", err)
		}

		if destinationGrant != nil {
			// Check if requested grant is higher than existing permissions
			if destinationGrant.Permission == reqACL {
				log.Println("User already has the requested permissions for bucket:", bucketName)
				return nil
			}

			// Get Bucket
			b, err := bucket.GetBucket(bucketName)
			if err != nil {
				log.Println("Error getting bucket:", err)
				return fmt.Errorf("error getting bucket: %v", err)
			}

			// Update existing permissions
			destinationGrant.Permission = reqACL
			if err := bucket.UpdateGrant(bucket.UpdateGrantReq{
				BucketID:   b.ID,
				GranteeID:  authorization.ID,
				Permission: reqACL,
			}); err != nil {
				log.Println("Error updating user permissions:", err)
				return fmt.Errorf("error updating user permissions: %v", err)
			}
		} else {
			// Get Bucket
			b, err := bucket.GetBucket(bucketName)
			if err != nil {
				log.Println("Error getting bucket:", err)
				return fmt.Errorf("error getting bucket: %v", err)
			}
			// Create new grant
			err = bucket.NewGrant(bucket.NewGrantReq{
				BucketID:   b.ID,
				GranteeID:  authorization.ID,
				Permission: reqACL,
			})
		}

		return nil
	} else {
		log.Println("User does not have permission to modify ACLs")
		return fmt.Errorf("user does not have permission to modify ACLs")
	}
}
