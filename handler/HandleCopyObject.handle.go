package handler

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
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

	// Strip leading slash
	value = strings.TrimPrefix(value, "/")

	// Remove bucket prefix from source if present (e.g. "mybucket/key" -> "key")
	if strings.HasPrefix(value, bucketName+"/") {
		value = value[len(bucketName)+1:]
	}

	// Validate source key and prevent path traversal
	filePath, err := util.ValidateBucketPath(bucketName, value)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Invalid copy source key", "", "")
		return
	}

	objectMetadata, err := objects.GetObject(bucketName, value, nil)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error retrieving source metadata for "+value, "", "")
		return
	}
	if objectMetadata == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "Source metadata does not exist for "+value, "", "")
		return
	}

	// check if user has permissions to source file
	if !auth.UserHasPermissionToObject(session, grant, objectMetadata, bucketName, value, auth.WritePermission) {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		return
	}

	// Validate destination key
	destinationFilePath, err := util.ValidateBucketPath(bucketName, destination)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Invalid destination key", "", "")
		return
	}

	// Get the bucket from context (loaded by middleware)
	b := middleware.RetrieveBucket(r)
	if b == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "the requested bucket does not exist", "", "")
		return
	}

	// Copying a key onto itself: S3 allows it only to replace metadata. It
	// must never reach the copy below, which would truncate the file while
	// reading it.
	if value == destination {
		if !strings.EqualFold(r.Header.Get("x-amz-metadata-directive"), "REPLACE") {
			responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "This copy request is illegal because it is trying to copy an object to itself without changing the object's metadata, storage class, website redirect location or encryption attributes.", requestId, hostId)
			return
		}
		responder.SendXML(w, http.StatusOK, &types.CopyObjectResult{
			ETag:         etagOf(objectMetadata),
			LastModified: objectMetadata.LastModified,
		})
		return
	}

	// ⚠️ A folder marker ("a/") is a directory on disk, not a file: reading it
	// failed with "is a directory", so every folder rename in openbucket-api
	// failed (#22). Copy it as what it is — a folder marker at the destination.
	if strings.HasSuffix(value, "/") {
		if !strings.HasSuffix(destination, "/") {
			responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "A folder can only be copied to a key ending in '/'", requestId, hostId)
			return
		}
		if err := os.MkdirAll(destinationFilePath, os.ModePerm); err != nil {
			responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error creating destination folder", requestId, hostId)
			return
		}
		if err := objects.CreateFolder(destinationFilePath, destination, *b, *session); err != nil {
			responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error creating destination folder", requestId, hostId)
			return
		}
		folder, err := objects.GetObject(bucketName, destination, nil, &objects.GetObjectOptions{})
		if err != nil || folder == nil {
			responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error reading destination folder", requestId, hostId)
			return
		}
		responder.SendXML(w, http.StatusOK, &types.CopyObjectResult{ETag: etagOf(folder), LastModified: folder.LastModified})
		return
	}

	// check if source file exists
	if info, err := os.Stat(filePath); os.IsNotExist(err) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "Source file does not exist", "", "")
		return
	} else if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error checking source file", "", "")
		return
	} else if info.IsDir() {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest", "Copy source is a folder; folder keys end in '/'", requestId, hostId)
		return
	}

	// An existing destination is overwritten, as in S3 (it used to be refused
	// with ObjectAlreadyExists, so moving a file onto an existing name failed,
	// #23). CreateObject replaces the file and upserts its metadata row.
	srcFile, err := os.Open(filePath)
	if err != nil {
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError", "Error reading source file", "", "")
		return
	}
	defer srcFile.Close()

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

func etagOf(o *types.ObjectMetadata) types.ETag {
	if o == nil || o.ETag == nil {
		return ""
	}
	return *o.ETag
}
