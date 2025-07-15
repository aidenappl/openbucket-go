package routers

import (
	"net/http"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/gorilla/mux"
)

func HandleCreateBucket(w http.ResponseWriter, r *http.Request) {
	// Get the bucket name from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Validate bucket name
	if bucket == "" {
		http.Error(w, "Bucket name must be provided", http.StatusBadRequest)
		return
	}

	if err := handler.CreateBucket(bucket); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Bucket created successfully"))
}
