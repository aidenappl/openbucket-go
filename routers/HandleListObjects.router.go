package routers

import (
	"encoding/xml"
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/gorilla/mux"
)

func HandleListObjects(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectList, err := handler.ListObjectsXML(bucket, r.URL.Query())
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to list objects", "", "")
		log.Println("Error listing objects:", err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")

	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(objectList)
}
