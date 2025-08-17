package routers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleUpload(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	if bucket == "" || key == "" {
		http.Error(w, "Bucket and key must be provided", http.StatusBadRequest)
		log.Println("Bucket or key is empty")
		return
	}

	user := middleware.RetrieveSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Unauthorized access attempt")
		return
	}

	filePath := filepath.Join("buckets", bucket, key)

	bucketDir := filepath.Join("buckets", bucket)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		http.Error(w, "Bucket not found", http.StatusNotFound)
		log.Println("Bucket not found:", bucketDir)
		return
	} else if err != nil {
		http.Error(w, "Unable to access bucket", http.StatusInternalServerError)
		log.Println("Error accessing bucket:", err)
		return
	}

	// Check different subrequest types
	found, _, _ := tools.HeaderExists(r, "x-amz-copy-source")
	if found {
		handler.HandleCopyObject(w, r)
		return
	}
	q := r.URL.Query()
	if _, ok := q["partNumber"]; ok {
		log.Println("Currently do not support multipart uploads")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}
	if _, ok := q["uploadId"]; ok {
		log.Println("Currently do not support multipart uploads")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}
	if _, ok := q["acl"]; ok {
		log.Println("Currently do not support acl")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}
	if _, ok := q["tagging"]; ok {
		log.Println("Currently do not support tagging")
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}
	if _, ok := q["legal-hold"]; ok {
		log.Println("LegalHold query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "LegalHold query parameter is not supported", "", "")
		return
	}
	if _, ok := q["retention"]; ok {
		log.Println("Retention query parameter is not supported for bucket operations")
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Retention query parameter is not supported", "", "")
		return
	}

	isDirectory := strings.HasSuffix(key, "/")
	if isDirectory {

		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			log.Println("Error creating directory:", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		log.Println("Directory created:", filePath)
		return
	}

	// Create File
	eTag, err := objects.CreateObject(filePath, key, bucket, r.Body, nil, &types.UserObject{
		ID:          user.KeyID,
		DisplayName: user.Name,
	})
	if err != nil {
		http.Error(w, "Failed to create object", http.StatusInternalServerError)
		log.Println("Error creating object:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Println("File uploaded successfully. ETag:", *eTag)
}
