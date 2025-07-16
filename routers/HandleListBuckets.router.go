package routers

import (
	"encoding/xml"
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/handler"
)

func HandleListBuckets(w http.ResponseWriter, r *http.Request) {
	// Read the buckets directory to list all bucket directories

	bucketsList, err := handler.ListBuckets()
	if err != nil {
		http.Error(w, "Failed to list buckets", http.StatusInternalServerError)
		log.Println("Error listing buckets:", err)
		return
	}

	// Set the Content-Type to XML
	w.Header().Set("Content-Type", "application/xml")
	// Write the response as XML
	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(bucketsList)
}
