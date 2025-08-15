package routers

import (
	"log"
	"net/http"
)

func HandleDeleteBucket(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Delete bucket operation is not implemented yet", http.StatusNotImplemented)
	log.Println("Delete bucket operation is not implemented yet")
}
