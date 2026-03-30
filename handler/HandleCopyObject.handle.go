package handler

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleCopyObject(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	destination := vars["key"]

	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)

	// Validate object metadata
	session := middleware.RetrieveSession(r)
	grant := middleware.RetrieveGrant(r)

	found, _, value := tools.HeaderExists(r, "x-amz-copy-source")
	if !found {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Missing Copy Source", "", "")
		return
	}

	// unescape copy source
	value, err := url.PathUnescape(value)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Failed to unescape Copy Source", "", "")
		return
	}

	// remove bucket from source if present
	if len(value) > len(bucketName) && value[0:len(bucketName)] == bucketName {
		value = value[len(bucketName)+1:]
	}

	filePath := filepath.Join("buckets", bucketName, value)

	objectMetadata, err := objects.GetObject(bucketName, value, nil)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error retrieving source metadata for "+value, "", "")
		return
	}
	if objectMetadata == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "Source metadata does not exist for "+value, "", "")
		return
	}

	// check if source file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "Source file does not exist", "", "")
		return
	} else if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error checking source file", "", "")
		return
	}

	// check if user has permissions to source file
	if !auth.UserHasPermissionToObject(session, grant, objectMetadata, bucketName, value, auth.WritePermission) {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		return
	}

	destinationFilePath := filepath.Join("buckets", bucketName, destination)

	// check if destination file already exists
	if _, err := os.Stat(destinationFilePath); !os.IsNotExist(err) {
		responder.SendXMLError(w, http.StatusConflict, "ObjectAlreadyExists", "Destination file already exists", "", "")
		return
	}

	// if all checks pass, proceed with the copy
	srcFile, err := os.Open(filePath)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error reading source file", "", "")
		return
	}
	defer srcFile.Close()

	// Get the bucket from context (loaded by middleware)
	b := middleware.RetrieveBucket(r)
	if b == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "the requested bucket does not exist", "", "")
		return
	}

	etag, err := objects.CreateObject(destinationFilePath, destination, *b, srcFile, nil, session, &objectMetadata.Public)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error creating destination object", "", "")
		return
	}
	log.Println("Object copied successfully. ETag:", *etag)

	responder.SendXML(w, http.StatusOK, &types.CopyObjectResult{
		ETag:         *etag,
		LastModified: objectMetadata.LastModified,
	})
}
