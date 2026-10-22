package routers

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aidenappl/openbucket-go/aws"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/gorilla/mux"
)

// S3 multipart upload: POST ?uploads (create), PUT ?partNumber&uploadId
// (upload part), POST ?uploadId (complete), DELETE ?uploadId (abort). Storage
// is in objects/Multipart.objects.go. ListParts / ListMultipartUploads are not
// implemented; rclone and the AWS SDKs' upload managers don't need them.

const s3Xmlns = "http://s3.amazonaws.com/doc/2006-03-01/"

type initiateMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

type completeMultipartUpload struct {
	XMLName xml.Name `xml:"CompleteMultipartUpload"`
	Parts   []struct {
		PartNumber int    `xml:"PartNumber"`
		ETag       string `xml:"ETag"`
	} `xml:"Part"`
}

type completeMultipartUploadResult struct {
	XMLName  xml.Name   `xml:"CompleteMultipartUploadResult"`
	Xmlns    string     `xml:"xmlns,attr"`
	Location string     `xml:"Location"`
	Bucket   string     `xml:"Bucket"`
	Key      string     `xml:"Key"`
	ETag     types.ETag `xml:"ETag"`
}

// multipartTarget resolves the bucket, key and session every multipart
// request needs, answering the error itself when it can't.
func multipartTarget(w http.ResponseWriter, r *http.Request) (*types.Bucket, string, *types.Authorization, bool) {
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key, err := url.PathUnescape(vars["key"])
	if err != nil || bucketName == "" || key == "" {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidBucketOrKey", "Bucket and key must be provided", requestId, hostId)
		return nil, "", nil, false
	}
	if err := util.ValidateObjectKey(key); err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidKey", err.Error(), requestId, hostId)
		return nil, "", nil, false
	}
	user := middleware.RetrieveSession(r)
	if user == nil {
		responder.SendAccessDeniedXML(w, &requestId, &hostId)
		return nil, "", nil, false
	}
	bucket := middleware.RetrieveBucket(r)
	if bucket == nil || !util.BucketExists(bucketName) {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "the requested bucket does not exist", requestId, hostId)
		return nil, "", nil, false
	}
	return bucket, key, user, true
}

func sendMultipartError(w http.ResponseWriter, r *http.Request, err error) {
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)
	code, msg := multipartErrorCode(err)
	status := http.StatusBadRequest
	switch code {
	case "NoSuchUpload":
		status = http.StatusNotFound
	case "InternalError":
		status = http.StatusInternalServerError
		log.Println(requestId, "multipart:", err)
	}
	responder.SendXMLError(w, status, code, msg, requestId, hostId)
}

// multipartErrorCode maps an error to its S3 error code and message.
func multipartErrorCode(err error) (string, string) {
	switch {
	case errors.Is(err, objects.ErrNoSuchUpload):
		return "NoSuchUpload", err.Error()
	case errors.Is(err, objects.ErrInvalidPart):
		return "InvalidPart", err.Error()
	case errors.Is(err, objects.ErrInvalidPartOrder):
		return "InvalidPartOrder", err.Error()
	case errors.Is(err, objects.ErrEntityTooSmall):
		return "EntityTooSmall", err.Error()
	case errors.Is(err, objects.ErrEntityTooLarge):
		return "EntityTooLarge", err.Error()
	case errors.Is(err, objects.ErrMalformedParts):
		return "InvalidArgument", err.Error()
	case errors.Is(err, objects.ErrBadDigest):
		return "BadDigest", "The Content-MD5 you specified did not match what we received."
	case errors.Is(err, aws.ErrChunkedChecksum):
		return "BadDigest", "The trailing checksum you specified did not match what we received."
	case errors.Is(err, aws.ErrChunkedFraming), errors.Is(err, aws.ErrChunkedLength):
		return "IncompleteBody", "The aws-chunked request body was malformed or incomplete."
	}
	return "InternalError", "We encountered an internal error. Please try again."
}

func writeXML(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(v); err != nil {
		log.Println("multipart: write response:", err)
	}
}

// handleCreateMultipartUpload: POST /{bucket}/{key}?uploads
func handleCreateMultipartUpload(w http.ResponseWriter, r *http.Request) {
	bucket, key, user, ok := multipartTarget(w, r)
	if !ok {
		return
	}
	public, ok := uploadACL(w, r)
	if !ok {
		return
	}
	up, err := objects.CreateMultipartUpload(*bucket, key, user, public)
	if err != nil {
		sendMultipartError(w, r, err)
		return
	}
	log.Println("multipart: created", up.ID, "for", bucket.Name+"/"+key)
	writeXML(w, initiateMultipartUploadResult{Xmlns: s3Xmlns, Bucket: bucket.Name, Key: key, UploadID: up.ID})
}

// handleUploadPart: PUT /{bucket}/{key}?partNumber=N&uploadId=ID
func handleUploadPart(w http.ResponseWriter, r *http.Request) {
	bucket, key, _, ok := multipartTarget(w, r)
	if !ok {
		return
	}
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)
	q := r.URL.Query()
	n, err := strconv.Atoi(q.Get("partNumber"))
	if err != nil || n < 1 || n > objects.MaxPartNumber {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidArgument", "Part number must be an integer between 1 and 10000, inclusive", requestId, hostId)
		return
	}
	up, err := objects.LoadMultipartUpload(q.Get("uploadId"), bucket.Name, key)
	if err != nil {
		sendMultipartError(w, r, err)
		return
	}
	var wantMD5 []byte
	if v := r.Header.Get("Content-MD5"); v != "" {
		sum, err := base64.StdEncoding.DecodeString(v)
		if err != nil || len(sum) != md5.Size {
			responder.SendXMLError(w, http.StatusBadRequest, "InvalidDigest", "The Content-MD5 you specified was invalid.", requestId, hostId)
			return
		}
		wantMD5 = sum
	}
	var body io.Reader = r.Body
	if aws.IsAWSChunked(r) {
		body = aws.NewChunkedReader(r)
	}
	etag, err := objects.WriteMultipartPart(up, n, body, wantMD5)
	if err != nil {
		sendMultipartError(w, r, err)
		return
	}
	w.Header().Set("ETag", etag.Quoted())
	w.WriteHeader(http.StatusOK)
}

// handleCompleteMultipartUpload: POST /{bucket}/{key}?uploadId=ID
func handleCompleteMultipartUpload(w http.ResponseWriter, r *http.Request) {
	bucket, key, user, ok := multipartTarget(w, r)
	if !ok {
		return
	}
	requestId := middleware.GetRequestID(r)
	hostId := middleware.GetHostID(r)
	up, err := objects.LoadMultipartUpload(r.URL.Query().Get("uploadId"), bucket.Name, key)
	if err != nil {
		sendMultipartError(w, r, err)
		return
	}
	var req completeMultipartUpload
	// 10,000 parts at ~100 bytes each.
	if err := xml.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "MalformedXML", "The XML you provided was not well-formed or did not validate against our published schema.", requestId, hostId)
		return
	}
	parts := make([]objects.CompletedPart, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = objects.CompletedPart{PartNumber: p.PartNumber, ETag: p.ETag}
	}
	filePath := filepath.Join("buckets", bucket.Name, key)

	// Validation is quick and its errors keep their status codes: SDKs treat a
	// 200 carrying an <Error> as a transient 500 and retry it.
	completion, err := objects.PrepareMultipartComplete(up, parts)
	if err != nil {
		sendMultipartError(w, r, err)
		return
	}

	// Joining gigabytes of parts can outlast Cloudflare's 100s and nginx's
	// 60s read timeouts. Like S3, answer 200 at once and send whitespace until
	// the result is ready; a failure while joining is an <Error> in the 200
	// body, which the SDKs and rclone check for and retry.
	type result struct {
		etag *types.ETag
		err  error
	}
	done := make(chan result, 1)
	go func() {
		etag, err := completion.Commit(filePath, *bucket, user)
		done <- result{etag, err}
	}()

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_ = rc.Flush()
	keepAlive := time.NewTicker(completeKeepAlive)
	defer keepAlive.Stop()
	for {
		select {
		case <-keepAlive.C:
			_, _ = w.Write([]byte(" "))
			_ = rc.Flush()
		case res := <-done:
			if res.err != nil {
				code, msg := multipartErrorCode(res.err)
				log.Println(requestId, "multipart: complete", up.ID, "failed:", res.err)
				_ = xml.NewEncoder(w).Encode(responder.ErrorResponse{Code: code, Message: msg, RequestId: requestId, HostId: hostId})
				return
			}
			log.Println("multipart: completed", up.ID, "as", bucket.Name+"/"+key, "ETag", *res.etag)
			_ = xml.NewEncoder(w).Encode(completeMultipartUploadResult{
				Xmlns:    s3Xmlns,
				Location: "/" + bucket.Name + "/" + key,
				Bucket:   bucket.Name,
				Key:      key,
				ETag:     *res.etag,
			})
			return
		}
	}
}

// completeKeepAlive is how often CompleteMultipartUpload sends whitespace
// while it joins the parts.
var completeKeepAlive = 10 * time.Second

// handleAbortMultipartUpload: DELETE /{bucket}/{key}?uploadId=ID
func handleAbortMultipartUpload(w http.ResponseWriter, r *http.Request) {
	bucket, key, _, ok := multipartTarget(w, r)
	if !ok {
		return
	}
	up, err := objects.LoadMultipartUpload(r.URL.Query().Get("uploadId"), bucket.Name, key)
	if err != nil {
		sendMultipartError(w, r, err)
		return
	}
	if err := objects.AbortMultipartUpload(up); err != nil {
		sendMultipartError(w, r, err)
		return
	}
	log.Println("multipart: aborted", up.ID)
	w.WriteHeader(http.StatusNoContent)
}

// uploadACL reads x-amz-acl the way a PUT does: private or public-read.
func uploadACL(w http.ResponseWriter, r *http.Request) (bool, bool) {
	switch acl := r.Header.Get("X-Amz-Acl"); acl {
	case "", "private":
		return false, true
	case "public-read":
		return true, true
	default:
		responder.SendXMLError(w, http.StatusNotImplemented, "NotImplemented", "ACL '"+acl+"' is not supported", middleware.GetRequestID(r), middleware.GetHostID(r))
		return false, false
	}
}
