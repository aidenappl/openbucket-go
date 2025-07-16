package routers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	if bucket == "" || key == "" {
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest", "Bucket and key must be provided", "", "")
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	permissions := middleware.RetrievePermissions(r)

	metadata := middleware.RetrieveMetadata(r)

	session := middleware.RetrieveSession(r)

	if !metadata.Public && !permissions.AllowGlobalRead && session == nil {
		if !isValidPresignURL(r, bucket, key) {
			responder.SendXML(w, http.StatusUnauthorized, "InvalidSignature", "The presigned URL is invalid or expired", "", "")
			log.Println(request, host, "Invalid or expired presigned URL:", key)
			return
		}
	}

	filePath := filepath.Join("buckets", bucket, key)

	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		responder.SendXML(w, http.StatusNotFound, "AccessDenied", "Access Denied", "", "")
		return
	} else if err != nil {

		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to retrieve file", "", "")
		log.Println(request, host, "Error opening file:", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Unable to get file info", "", "")
		log.Println(request, host, "Error getting file info:", err)
		return
	}

	w.Header().Set("ETag", metadata.ETag)
	w.Header().Set("Content-Type", tools.ContentType(filePath))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))

	_, err = io.Copy(w, file)
	if err != nil {

		responder.SendXML(w, http.StatusInternalServerError, "InternalError", "Error transferring file", "", "")
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
