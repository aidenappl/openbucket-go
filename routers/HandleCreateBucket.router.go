package routers

import (
	"net/http"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/gorilla/mux"
)

func HandleCreateBucket(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Check if any ACL headers are present
	aclHeaders := map[string]func(string){
		"x-amz-grant-full-control": handleNothing,
		"x-amz-grant-read":         handleNothing,
		"x-amz-grant-write":        handleNothing,
		"x-amz-grant-read-acp":     handleNothing,
		"x-amz-grant-write-acp":    handleNothing,
	}

	var found bool
	for name, handler := range aclHeaders {
		if v := r.Header.Get(name); v != "" {
			found = true
			handler(v) // comment out if you truly don't support ACLs yet
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

func handleNothing(value string) {
	// This function intentionally does nothing.
	// It is a placeholder for handling ACL headers if needed in the future.
}
