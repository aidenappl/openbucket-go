package routers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

type createGrantRequest struct {
	KeyID      string `json:"key_id"`
	Permission string `json:"permission"`
}

type updateGrantRequest struct {
	Permission string `json:"permission"`
}

// HandleAdminListGrants returns all grants for a bucket.
func HandleAdminListGrants(w http.ResponseWriter, r *http.Request) {
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

	out := make([]grantSummary, len(b.Grants))
	for i, g := range b.Grants {
		out[i] = grantSummary{
			ID:          g.ID,
			KeyID:       g.Grantee.ID,
			DisplayName: g.Grantee.DisplayName,
			Permission:  string(g.Permission),
			DateAdded:   g.DateAdded.Format("2006-01-02T15:04:05Z"),
		}
	}

	responder.SendJSON(w, http.StatusOK, out)
}

// HandleAdminCreateGrant adds a grant to a bucket.
func HandleAdminCreateGrant(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucket"]

	var req createGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.KeyID == "" {
		responder.SendJSONError(w, http.StatusBadRequest, "key_id is required")
		return
	}

	if err := handler.GrantAccess(bucketName, req.KeyID, req.Permission); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	bucket.InvalidateBucketCache(bucketName)
	auth.InvalidatePermCacheForBucket(bucketName)
	responder.SendJSON(w, http.StatusCreated, nil)
}

// HandleAdminUpdateGrant updates a grant's permission level.
func HandleAdminUpdateGrant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	keyID := vars["keyId"]

	var req updateGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Permission == "" {
		responder.SendJSONError(w, http.StatusBadRequest, "permission is required")
		return
	}

	perm := types.ConvertToPermission(req.Permission)
	if perm == types.ACLUnknown {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid permission type")
		return
	}

	if err := handler.UpdateGrantAccess(bucketName, keyID, perm); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	bucket.InvalidateBucketCache(bucketName)
	auth.InvalidatePermCacheForBucket(bucketName)
	responder.SendJSON(w, http.StatusOK, nil)
}

// HandleAdminDeleteGrant removes a grant by ID.
func HandleAdminDeleteGrant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid grant id")
		return
	}

	if err := bucket.DeleteGrant(id); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	bucket.InvalidateBucketCache(bucketName)
	auth.InvalidatePermCacheForBucket(bucketName)
	responder.SendJSON(w, http.StatusOK, nil)
}
