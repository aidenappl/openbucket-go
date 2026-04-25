package routers

import (
	"encoding/json"
	"net/http"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

type adminBucketResponse struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	CreationDate string         `json:"creation_date"`
	ACL          string         `json:"acl"`
	Owner        ownerResponse  `json:"owner"`
	Grants       []grantSummary `json:"grants,omitempty"`
}

type ownerResponse struct {
	KeyID       string `json:"key_id"`
	DisplayName string `json:"display_name"`
}

type grantSummary struct {
	ID          int    `json:"id"`
	KeyID       string `json:"key_id"`
	DisplayName string `json:"display_name"`
	Permission  string `json:"permission"`
	DateAdded   string `json:"date_added"`
}

type createBucketRequest struct {
	Name       string `json:"name"`
	OwnerKeyID string `json:"owner_key_id"`
}

type updateBucketACLRequest struct {
	ACL string `json:"acl"`
}

// HandleAdminListBuckets returns all buckets with their grants.
func HandleAdminListBuckets(w http.ResponseWriter, r *http.Request) {
	includeGrants := r.URL.Query().Get("grants") == "true"

	buckets, err := bucket.ListBuckets(&bucket.ListBucketsParams{
		Include: &bucket.ListBucketsParamInclude{
			Grants: includeGrants,
		},
	})
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]adminBucketResponse, len(*buckets))
	for i, b := range *buckets {
		resp := adminBucketResponse{
			ID:           b.ID,
			Name:         b.Name,
			CreationDate: b.CreationDate.Format("2006-01-02T15:04:05Z"),
			ACL:          string(b.ACL),
			Owner: ownerResponse{
				KeyID:       b.Owner.ID,
				DisplayName: b.Owner.DisplayName,
			},
		}
		for _, g := range b.Grants {
			resp.Grants = append(resp.Grants, grantSummary{
				ID:          g.ID,
				KeyID:       g.Grantee.ID,
				DisplayName: g.Grantee.DisplayName,
				Permission:  string(g.Permission),
				DateAdded:   g.DateAdded.Format("2006-01-02T15:04:05Z"),
			})
		}
		out[i] = resp
	}

	responder.SendJSON(w, http.StatusOK, out)
}

// HandleAdminGetBucket returns a single bucket with grants.
func HandleAdminGetBucket(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucket"]

	b, err := bucket.GetBucket(bucketName)
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if b == nil {
		responder.SendJSONError(w, http.StatusNotFound, "bucket not found")
		return
	}

	resp := adminBucketResponse{
		ID:           b.ID,
		Name:         b.Name,
		CreationDate: b.CreationDate.Format("2006-01-02T15:04:05Z"),
		ACL:          string(b.ACL),
		Owner: ownerResponse{
			KeyID:       b.Owner.ID,
			DisplayName: b.Owner.DisplayName,
		},
	}
	for _, g := range b.Grants {
		resp.Grants = append(resp.Grants, grantSummary{
			ID:          g.ID,
			KeyID:       g.Grantee.ID,
			DisplayName: g.Grantee.DisplayName,
			Permission:  string(g.Permission),
			DateAdded:   g.DateAdded.Format("2006-01-02T15:04:05Z"),
		})
	}

	responder.SendJSON(w, http.StatusOK, resp)
}

// HandleAdminCreateBucket creates a new bucket owned by the given credential.
func HandleAdminCreateBucket(w http.ResponseWriter, r *http.Request) {
	var req createBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.OwnerKeyID == "" {
		responder.SendJSONError(w, http.StatusBadRequest, "name and owner_key_id are required")
		return
	}

	owner, err := auth.CheckUserExists(req.OwnerKeyID)
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if owner == nil {
		responder.SendJSONError(w, http.StatusBadRequest, "owner credential not found")
		return
	}

	if err := bucket.CreateBucket(req.Name, owner.ID); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch the newly created bucket to return it
	b, err := bucket.GetBucket(req.Name)
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := adminBucketResponse{
		ID:           b.ID,
		Name:         b.Name,
		CreationDate: b.CreationDate.Format("2006-01-02T15:04:05Z"),
		ACL:          string(b.ACL),
		Owner: ownerResponse{
			KeyID:       b.Owner.ID,
			DisplayName: b.Owner.DisplayName,
		},
	}

	responder.SendJSON(w, http.StatusCreated, resp)
}

// HandleAdminDeleteBucket deletes a bucket and all its data.
func HandleAdminDeleteBucket(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucket"]

	if err := bucket.DeleteBucket(bucketName); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	bucket.InvalidateBucketCache(bucketName)
	auth.InvalidatePermCacheForBucket(bucketName)
	responder.SendJSON(w, http.StatusOK, nil)
}

// HandleAdminGetBucketStats returns object count and total size for a bucket.
func HandleAdminGetBucketStats(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucket"]

	// Verify bucket exists
	b, err := bucket.GetBucket(bucketName, &bucket.GetBucketOptions{IncludeGrants: false})
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if b == nil {
		responder.SendJSONError(w, http.StatusNotFound, "bucket not found")
		return
	}

	stats, err := bucket.GetBucketStats(bucketName)
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responder.SendJSON(w, http.StatusOK, stats)
}

// HandleAdminUpdateBucketACL sets the bucket-level ACL.
func HandleAdminUpdateBucketACL(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucket"]

	var req updateBucketACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ACL == "" {
		responder.SendJSONError(w, http.StatusBadRequest, "acl is required")
		return
	}

	aclType := types.ConvertToBucketACL(req.ACL)
	if aclType == "" {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid ACL type")
		return
	}

	b, err := bucket.GetBucket(bucketName, &bucket.GetBucketOptions{IncludeGrants: false})
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if b == nil {
		responder.SendJSONError(w, http.StatusNotFound, "bucket not found")
		return
	}

	if err := bucket.UpdateBucketACL(b.ID, aclType); err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	bucket.InvalidateBucketCache(bucketName)
	responder.SendJSON(w, http.StatusOK, nil)
}
