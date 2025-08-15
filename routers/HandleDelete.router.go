package routers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	filePath := filepath.Join("buckets", bucket, key)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Object not found", http.StatusNotFound)
		log.Println("Object not found:", filePath)
		return
	}

	q := r.URL.Query()
	if _, ok := q["uploadId"]; ok {
		log.Println("Currently do not support multipart cancellations")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}
	if _, ok := q["tagging"]; ok {
		log.Println("Currently do not support tagging")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	err := os.Remove(filePath)
	if err != nil {
		http.Error(w, "Failed to delete object", http.StatusInternalServerError)
		log.Println("Error deleting file:", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	log.Printf("Successfully deleted object %s from bucket %s", key, bucket)
}
