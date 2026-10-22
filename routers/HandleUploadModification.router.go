package routers

import (
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/responder"
)

func HandleUploadModification(w http.ResponseWriter, r *http.Request) {
	// Sub Query Verification
	q := r.URL.Query()
	if _, ok := q["uploads"]; ok {
		handleCreateMultipartUpload(w, r)
		return
	}
	if _, ok := q["uploadId"]; ok {
		handleCompleteMultipartUpload(w, r)
		return
	}
	if _, ok := q["restore"]; ok {
		log.Println("Restore query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Restore query parameter is not supported", "", "")
		return
	}
	if _, ok := q["select"]; ok {
		log.Println("Select query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Select query parameter is not supported", "", "")
		return
	}
}
