package routers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	if bucket == "" || key == "" {
		responder.SendAccessDeniedXML(w, nil, nil)
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	filePath := filepath.Join("buckets", bucket, key)
	file, err := os.Open(filePath)
	if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error getting file info:", err)
		return
	}
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		responder.SendAccessDeniedXML(w, &request, &host)
		return
	} else if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error opening file:", err)
		return
	} else if fileInfo.IsDir() {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "File is a directory, not a valid object:", filePath)
		return
	}
	defer file.Close()

	permissions := middleware.RetrievePermissions(r)
	metadata := middleware.RetrieveMetadata(r)
	session := middleware.RetrieveSession(r)

	if !metadata.Public && !types.IsBucketACLRead(permissions.ACL) && session == nil {
		if !isValidPresignURL(r, bucket, key) {
			responder.SendAccessDeniedXML(w, &request, &host)
			log.Println(request, host, "Invalid or expired presigned URL:", key)
			return
		}

		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Access denied for bucket:", bucket, "key:", key)
		return
	}

	w.Header().Set("ETag", metadata.ETag)
	w.Header().Set("X-Amz-Meta-owner-id", metadata.Owner)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	w.Header().Set("Content-Type", tools.ContentType(filePath))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("x-amz-tagging-count", strconv.Itoa(len(metadata.Tags)))
	w.Header().Set("x-amz-version-id", metadata.VersionID)

	_, err = io.Copy(w, file)
	if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error transferring file:", err)
		return
	}

	log.Println("File successfully served:", filePath)
}

func isValidPresignURL(r *http.Request, bucket, key string) bool {
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)
	amzDate := r.URL.Query().Get("X-Amz-Date")
	signature := r.URL.Query().Get("X-Amz-Signature")
	if amzDate == "" || signature == "" {
		log.Println(request, host, "Missing required parameters for presigned URL validation")
		return false
	}
	return true
}
