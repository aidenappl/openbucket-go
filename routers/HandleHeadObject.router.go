package routers

import (
	"encoding/xml"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aidenappl/openbucket-go/metadata"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/gorilla/mux"
)

func HandleHeadObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	rawKey := vars["key"]

	if bucket == "" || rawKey == "" {
		responder.SendXML(w, http.StatusBadRequest, "InvalidRequest",
			"Bucket and key must be provided", "", "")
		return
	}

	cleanKey := path.Clean("/" + rawKey)
	if strings.Contains(cleanKey, "..") ||
		strings.HasSuffix(cleanKey, "/") {
		responder.SendXML(w, http.StatusBadRequest, "InvalidKey",
			"Invalid object key", "", "")
		return
	}

	objPath := filepath.Join("buckets", bucket, cleanKey)
	info, err := os.Stat(objPath)
	if err != nil {
		responder.SendXML(w, http.StatusNotFound, "NoSuchKey",
			"Object not found", "", "")
		return
	}

	if info.IsDir() {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		return
	}

	var meta metadata.Metadata
	metaPath := objPath + ".obmeta"
	if f, err := os.Open(metaPath); err == nil {
		_ = xml.NewDecoder(f).Decode(&meta)
		f.Close()
	}

	cType := tools.ContentType(objPath)

	w.Header().Set("Content-Type", cType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	if meta.ETag != "" {
		w.Header().Set("ETag", meta.ETag)
	}

	w.WriteHeader(http.StatusOK)
}
