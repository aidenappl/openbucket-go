package routers

import (
	"fmt"
	"net/http"

	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

func HandlePresignURL(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	expiration := int64(900)
	signedURL := tools.GeneratePresignedURL(bucket, key, expiration)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"url": "%s"}`, signedURL)))
}
