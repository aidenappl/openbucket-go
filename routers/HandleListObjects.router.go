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
	// Get the bucket name from the URL
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectList, err := handler.ListObjectsXML(bucket)
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to list objects", "", "")
		log.Println("Error listing objects:", err)
		return
	}

	// Set the response header to XML
	w.Header().Set("Content-Type", "application/xml")
	// Write the XML response
	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(objectList)
}
