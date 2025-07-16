package routers

import (
	"fmt"
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
		return
	}

	err := os.Remove(filePath)
	if err != nil {
		http.Error(w, "Failed to delete object", http.StatusInternalServerError)
		log.Println("Error deleting file:", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	fmt.Printf("Deleted object: %s\n", filePath)
}
