package routers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aidenappl/openbucket-go/aws"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/gorilla/mux"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {

	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	q := r.URL.Query()
	if _, ok := q["acl"]; ok {
		handleObjectACL(w, r)
		return
	}
	if _, ok := q["tagging"]; ok {
		handleGetObjectTagging(w, r)
		return
	}
	if _, ok := q["uploadId"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "UploadId query parameter is not supported for download")
		return
	}
	if _, ok := q["attributes"]; ok {
		handleGetObjectAttributes(w, r)
		return
	}
	if _, ok := q["legal-hold"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "LegalHold query parameter is not supported for download")
		return
	}
	if _, ok := q["retention"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Retention query parameter is not supported for download")
		return
	}
	if _, ok := q["torrent"]; ok {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Torrent query parameter is not supported for download")
		return
	}

	handleDownload(w, r)
}

func handleGetObjectTagging(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]
	// Get host & request from context
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Validate bucket & key
	if bucketName == "" || key == "" {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	// Get the object metadata
	metadata := middleware.RetrieveMetadata(r)
	if metadata == nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Object not found:", key)
		return
	}

	resp := types.GetObjectTaggingResponse{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
	}

	for _, tag := range metadata.Tags.Tag {
		miniTag := types.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		}
		resp.TagSet = append(resp.TagSet, miniTag)
	}

	responder.SendXML(w, http.StatusOK, resp)
}

func handleGetObjectAttributes(w http.ResponseWriter, r *http.Request) {
	// Get request variables
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]
	// Get host & request from context
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Validate bucket & key
	if bucketName == "" || key == "" {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	// Get the object metadata
	metadata := middleware.RetrieveMetadata(r)
	if metadata == nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Object not found:", key)
		return
	}

	// Build the response
	response := types.GetObjectAttributesResponse{
		Xmlns:        "http://s3.amazonaws.com/doc/2006-03-01/",
		ETag:         metadata.ETag.ToString(),
		ObjectSize:   metadata.Size,
		StorageClass: "STANDARD", // Currently do not support different storage classess
		Checksum: &types.Checksum{ // Currently do not support checksum
			CRC32:  "",
			SHA1:   "",
			SHA256: "",
			CRC32C: "",
		},
	}

	// Send success response
	responder.SendXML(w, http.StatusOK, response)
}

func handleObjectACL(w http.ResponseWriter, r *http.Request) {
	// Get request variables
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]

	// Get host & request from context
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Validate bucket & key
	if bucketName == "" || key == "" {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	// Get the bucket
	bucket := middleware.RetrieveBucket(r)
	if bucket == nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Bucket not found:", bucketName)
		return
	}

	// Retrieve object metadata to check for owner info
	metadata := middleware.RetrieveMetadata(r)
	if metadata == nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Object not found:", key)
		return
	}

	response := types.GetObjectACLResponse{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: bucket.Owner,
	}

	for _, grant := range bucket.Grants {
		miniGrant := types.MinifiedGrant{
			Grantee: types.Grantee{
				XmlnsXsi:    "http://www.w3.org/2001/XMLSchema-instance",
				XsiType:     "CanonicalUser",
				ID:          grant.Grantee.ID,
				DisplayName: grant.Grantee.DisplayName,
			},
			Permission: grant.Permission,
		}
		response.AccessControlList = append(response.AccessControlList, miniGrant)
	}

	publicGrant := types.MinifiedGrant{
		Grantee: types.Grantee{
			DisplayName: "All Users",
			XmlnsXsi:    "http://www.w3.org/2001/XMLSchema-instance",
			XsiType:     "Group",
			URI:         "http://openbucket/groups/global/AllUsers",
		},
		Permission: types.ConvertToBucketACL(tools.ACLString(metadata.Public)),
	}
	response.AccessControlList = append(response.AccessControlList, publicGrant)

	// Send the response
	responder.SendXML(w, http.StatusOK, response)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {

	// Get request variables
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]
	// Get host & request from context
	request := middleware.GetRequestID(r)
	host := middleware.GetHostID(r)

	// Validate bucket & key
	if bucket == "" || key == "" {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Bucket or key is empty")
		return
	}

	// Structure request
	filePath := filepath.Join("buckets", bucket, key)

	// Open requested file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			responder.SendXMLError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", request, host)
			return
		}
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error opening file:", err)
		return
	}
	defer file.Close()

	// Get file information from opened handle (avoids extra syscall)
	fileInfo, err := file.Stat()
	if err != nil {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "Error getting file info:", err)
		return
	}
	if fileInfo.IsDir() {
		responder.SendAccessDeniedXML(w, &request, &host)
		log.Println(request, host, "File is a directory, not a valid object:", filePath)
		return
	}

	// Check user permissions for file
	permissions := middleware.RetrieveBucket(r) // Bucket permissions
	metadata := middleware.RetrieveMetadata(r)  // File metadata
	session := middleware.RetrieveSession(r)    // User session

	if (metadata == nil || !metadata.Public) && (permissions == nil || !types.IsBucketACLRead(permissions.ACL)) && session == nil {
		// Check if request has a valid presigned URL
		if !isValidPresignURL(r, bucket, key) {
			responder.SendAccessDeniedXML(w, &request, &host)
			log.Println(request, host, "Access denied for bucket:", bucket, "key:", key)
			return
		}
		// Presigned URL is valid - continue to serve the file
	}

	// Structure response headers
	if metadata != nil {
		w.Header().Set("ETag", metadata.ETag.ToString())
		w.Header().Set("x-amz-meta-owner-id", metadata.Owner.ID)
		w.Header().Set("x-amz-meta-owner-display-name", metadata.Owner.DisplayName)
		w.Header().Set("x-amz-meta-acl", tools.ACLString(metadata.Public))
		w.Header().Set("x-amz-tagging-count", strconv.Itoa(len(metadata.Tags.Tag)))
		w.Header().Set("x-amz-version-id", strconv.Itoa(metadata.VersionID))
	}
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	w.Header().Set("Content-Type", tools.ContentType(filePath))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))

	// Transfer file content
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

	// Check for AWS presigned URL parameters
	q := r.URL.Query()
	amzDate := q.Get("X-Amz-Date")
	signature := q.Get("X-Amz-Signature")
	credential := q.Get("X-Amz-Credential")
	expires := q.Get("X-Amz-Expires")
	signedHeaders := q.Get("X-Amz-SignedHeaders")
	algorithm := q.Get("X-Amz-Algorithm")

	// Validate all required parameters exist
	if amzDate == "" || signature == "" || credential == "" || expires == "" || signedHeaders == "" {
		log.Println(request, host, "Missing required parameters for presigned URL validation")
		return false
	}

	// Validate algorithm
	if algorithm != "" && algorithm != "AWS4-HMAC-SHA256" {
		log.Println(request, host, "Invalid algorithm for presigned URL:", algorithm)
		return false
	}

	// Parse and validate expiration
	requestTime, err := time.Parse("20060102T150405Z", amzDate)
	if err != nil {
		log.Println(request, host, "Invalid X-Amz-Date format:", amzDate)
		return false
	}

	expiresSeconds, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		log.Println(request, host, "Invalid X-Amz-Expires value:", expires)
		return false
	}

	// Check if URL has expired
	expirationTime := requestTime.Add(time.Duration(expiresSeconds) * time.Second)
	if time.Now().After(expirationTime) {
		log.Println(request, host, "Presigned URL has expired")
		return false
	}

	// Validate the signature using AWS package
	if !aws.ValidatePresignedURLSignature(r, credential, signedHeaders, signature, amzDate) {
		log.Println(request, host, "Invalid presigned URL signature")
		return false
	}

	return true
}
