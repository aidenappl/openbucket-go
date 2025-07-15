package routers

import (
	"fmt"
	"net/http"

	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

func HandlePresignURL(w http.ResponseWriter, r *http.Request) {
	// Get the bucket and key (file name) from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	// Generate the pre-signed URL with an expiration of 15 minutes (900 seconds)
	expiration := int64(900)
	signedURL := tools.GeneratePresignedURL(bucket, key, expiration)

	// Return the pre-signed URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"url": "%s"}`, signedURL)))
}
