package routers

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
)

// HandleExportBucket streams a .tar.gz archive containing the full bucket:
//   - manifest.json  (bucket metadata, permissions, object metadata)
//   - objects/...     (file content, preserving key paths)
func HandleExportBucket(w http.ResponseWriter, r *http.Request, bucketName string) {
	hostID := middleware.GetHostID(r)
	reqID := middleware.GetRequestID(r)

	// Load bucket with grants
	b, err := bucket.GetBucket(bucketName)
	if err != nil || b == nil {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket",
			"The specified bucket does not exist", reqID, hostID)
		return
	}

	// Load bucket tags
	tags, err := bucket.GetBucketTags(b.ID)
	if err != nil {
		log.Println("export: error loading bucket tags:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError",
			"Failed to load bucket tags", reqID, hostID)
		return
	}

	// List all objects
	objs, err := objects.ListObjects(bucketName, &objects.ListObjectsOptions{BucketID: b.ID})
	if err != nil {
		log.Println("export: error listing objects:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError",
			"Failed to list objects", reqID, hostID)
		return
	}

	// Build manifest
	manifest := types.BackupManifest{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Bucket: types.BackupBucket{
			Name:         b.Name,
			ACL:          string(b.ACL),
			CreationDate: b.CreationDate,
		},
	}

	// Bucket tags
	if tags != nil {
		for _, t := range *tags {
			manifest.Bucket.Tags = append(manifest.Bucket.Tags, types.BackupTag{
				Key:   t.Key,
				Value: t.Value,
			})
		}
	}

	// Permissions — store by key_id so import can match
	for _, g := range b.Grants {
		manifest.Permissions = append(manifest.Permissions, types.BackupPermission{
			KeyID:      g.Grantee.ID,
			Permission: string(g.Permission),
		})
	}

	// Object metadata + tags
	for i, obj := range objs {
		meta := types.BackupObjectMetadata{
			Key:          obj.Key,
			ETag:         string(*obj.ETag),
			Size:         obj.Size,
			Public:       obj.Public,
			LastModified: time.Time(obj.LastModified),
			OwnerKeyID:   obj.Owner.ID,
		}

		objTags, err := objects.GetObjectTags(b.ID, obj.ID)
		if err != nil {
			log.Printf("export: error loading tags for object %s: %v", obj.Key, err)
		} else {
			for _, t := range objTags {
				meta.Tags = append(meta.Tags, types.BackupTag{Key: t.Key, Value: t.Value})
			}
		}

		manifest.Objects = append(manifest.Objects, meta)

		// Log progress for large exports
		if (i+1)%1000 == 0 {
			log.Printf("export: processed metadata for %d/%d objects", i+1, len(objs))
		}
	}

	// Stream the tar.gz response
	fileName := fmt.Sprintf("%s-export-%s.tar.gz", bucketName, time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.WriteHeader(http.StatusOK)

	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write manifest.json
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		log.Println("export: error marshaling manifest:", err)
		return
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:    "manifest.json",
		Size:    int64(len(manifestBytes)),
		Mode:    0644,
		ModTime: time.Now(),
	}); err != nil {
		log.Println("export: error writing manifest header:", err)
		return
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		log.Println("export: error writing manifest:", err)
		return
	}

	// Write each object file
	bucketPath := util.GetBucketPath(bucketName)
	for i, obj := range objs {
		filePath := filepath.Join(bucketPath, obj.Key)

		fi, err := os.Stat(filePath)
		if err != nil {
			log.Printf("export: skipping object %s: %v", obj.Key, err)
			continue
		}

		if fi.IsDir() {
			continue
		}

		if err := tw.WriteHeader(&tar.Header{
			Name:    filepath.Join("objects", obj.Key),
			Size:    fi.Size(),
			Mode:    0644,
			ModTime: fi.ModTime(),
		}); err != nil {
			log.Printf("export: error writing header for %s: %v", obj.Key, err)
			return
		}

		f, err := os.Open(filePath)
		if err != nil {
			log.Printf("export: error opening %s: %v", obj.Key, err)
			return
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			log.Printf("export: error writing %s: %v", obj.Key, err)
			return
		}
		f.Close()

		if (i+1)%1000 == 0 {
			log.Printf("export: archived %d/%d objects", i+1, len(objs))
		}
	}

	log.Printf("export: completed — bucket=%s objects=%d", bucketName, len(objs))
}
