package routers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/gorilla/mux"
)

type createCredentialRequest struct {
	Name string `json:"name"`
}

type credentialResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	KeyID       string `json:"key_id"`
	SecretKey   string `json:"secret_key,omitempty"`
	DateCreated string `json:"date_created"`
}

// HandleAdminListCredentials returns all credentials (secret keys are included).
func HandleAdminListCredentials(w http.ResponseWriter, r *http.Request) {
	auths, err := auth.LoadAuthorizations()
	if err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]credentialResponse, len(auths.Authorizations))
	for i, a := range auths.Authorizations {
		out[i] = credentialResponse{
			ID:          a.ID,
			Name:        a.Name,
			KeyID:       a.KeyID,
			DateCreated: a.DateCreated.Format("2006-01-02T15:04:05Z"),
		}
	}

	responder.SendJSON(w, http.StatusOK, out)
}

// HandleAdminCreateCredential generates a new KeyID/SecretKey pair and saves it.
func HandleAdminCreateCredential(w http.ResponseWriter, r *http.Request) {
	var req createCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		responder.SendJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	creds := handler.GenerateCredentials()
	creds.Name = req.Name

	if err := auth.SaveCredentials(creds); err != nil {
		responder.SendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responder.SendJSON(w, http.StatusCreated, credentialResponse{
		ID:          creds.ID,
		Name:        creds.Name,
		KeyID:       creds.KeyID,
		SecretKey:   creds.SecretKey,
		DateCreated: creds.DateCreated.Format("2006-01-02T15:04:05Z"),
	})
}

// HandleAdminDeleteCredential removes a credential by ID.
func HandleAdminDeleteCredential(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, "invalid credential id")
		return
	}

	if err := auth.DeleteAuthorization(id); err != nil {
		responder.SendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	responder.SendJSON(w, http.StatusOK, nil)
}
