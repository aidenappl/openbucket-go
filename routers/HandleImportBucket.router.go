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
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
)

const maxImportSize = 10 << 30 // 10 GB

// HandleImportBucket accepts a .tar.gz archive (produced by export) and
// recreates the bucket with all metadata, permissions, and object files.
//
// The importing user (from the request auth) becomes the bucket owner.
// Permissions are matched by key_id — if a key_id from the manifest exists
// on this instance, the grant is recreated; otherwise it is skipped with a
// warning in the response.
func HandleImportBucket(w http.ResponseWriter, r *http.Request) {
	hostID := middleware.GetHostID(r)
	reqID := middleware.GetRequestID(r)

	user := middleware.RetrieveSession(r)
	if user == nil {
		responder.SendXMLError(w, http.StatusForbidden, "AccessDenied",
			"Authentication required for import", reqID, hostID)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportSize)

	gr, err := gzip.NewReader(r.Body)
	if err != nil {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
			"Request body is not valid gzip", reqID, hostID)
		return
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// First pass: read manifest
	var manifest types.BackupManifest
	var foundManifest bool

	// Buffer files to a temp directory while we process
	tmpDir, err := os.MkdirTemp("", "openbucket-import-*")
	if err != nil {
		log.Println("import: error creating temp dir:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError",
			"Failed to create temp directory", reqID, hostID)
		return
	}
	defer os.RemoveAll(tmpDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
				"Invalid tar archive: "+err.Error(), reqID, hostID)
			return
		}

		// Validate path to prevent traversal
		clean := filepath.Clean(header.Name)
		if strings.Contains(clean, "..") {
			responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
				"Archive contains path traversal", reqID, hostID)
			return
		}

		if header.Name == "manifest.json" {
			data, err := io.ReadAll(io.LimitReader(tr, 50<<20)) // 50MB max manifest
			if err != nil {
				responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
					"Failed to read manifest.json", reqID, hostID)
				return
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
					"Invalid manifest.json: "+err.Error(), reqID, hostID)
				return
			}
			foundManifest = true
			continue
		}

		// Write object files to temp dir
		if strings.HasPrefix(header.Name, "objects/") && header.Typeflag == tar.TypeReg {
			destPath := filepath.Join(tmpDir, header.Name)
			if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
				log.Printf("import: error creating dir for %s: %v", header.Name, err)
				continue
			}
			f, err := os.Create(destPath)
			if err != nil {
				log.Printf("import: error creating file %s: %v", header.Name, err)
				continue
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				log.Printf("import: error writing file %s: %v", header.Name, err)
				continue
			}
			f.Close()
		}
	}

	if !foundManifest {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
			"Archive does not contain manifest.json", reqID, hostID)
		return
	}

	if manifest.Version != 1 {
		responder.SendXMLError(w, http.StatusBadRequest, "InvalidRequest",
			fmt.Sprintf("Unsupported manifest version: %d", manifest.Version), reqID, hostID)
		return
	}

	bucketName := manifest.Bucket.Name

	// Check if a rename was requested via query param
	if rename := r.URL.Query().Get("bucket"); rename != "" {
		bucketName = rename
	}

	// Check if bucket already exists
	existing, err := bucket.GetBucket(bucketName)
	if err != nil {
		log.Println("import: error checking bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError",
			"Failed to check bucket existence", reqID, hostID)
		return
	}
	if existing != nil {
		responder.SendXMLError(w, http.StatusConflict, "BucketAlreadyExists",
			fmt.Sprintf("Bucket %q already exists", bucketName), reqID, hostID)
		return
	}

	// Create the bucket (owner = importing user)
	if err := bucket.CreateBucket(bucketName, user.ID); err != nil {
		log.Println("import: error creating bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError",
			"Failed to create bucket: "+err.Error(), reqID, hostID)
		return
	}

	// Re-fetch to get the bucket ID
	b, err := bucket.GetBucket(bucketName)
	if err != nil || b == nil {
		log.Println("import: error fetching created bucket:", err)
		responder.SendXMLError(w, http.StatusInternalServerError, "InternalError",
			"Failed to fetch created bucket", reqID, hostID)
		return
	}

	// Set ACL
	acl := types.ConvertToPermission(manifest.Bucket.ACL)
	if acl != types.ACLUnknown {
		_, err := db.Psql.Update("buckets").
			Set("acl", string(acl)).
			Where("id = ?", b.ID).
			Exec()
		if err != nil {
			log.Printf("import: failed to set ACL: %v", err)
		}
	}

	// Bucket tags
	for _, t := range manifest.Bucket.Tags {
		if err := bucket.CreateBucketTag(b.ID, types.BucketTag{Key: t.Key, Value: t.Value}); err != nil {
			log.Printf("import: failed to create bucket tag %s: %v", t.Key, err)
		}
	}

	// Permissions — match by key_id
	var skippedPerms []string
	for _, p := range manifest.Permissions {
		// Skip if this is the owner (already granted FULL_CONTROL by CreateBucket)
		if p.KeyID == user.KeyID {
			continue
		}
		grantee, err := auth.LoadAuthorization(p.KeyID)
		if err != nil || grantee == nil {
			skippedPerms = append(skippedPerms, p.KeyID)
			log.Printf("import: skipping permission for unknown key_id %s", p.KeyID)
			continue
		}
		perm := types.ConvertToPermission(p.Permission)
		if perm == types.ACLUnknown {
			perm = types.READ
		}
		if err := bucket.NewGrant(bucket.NewGrantReq{
			BucketID:   b.ID,
			GranteeID:  grantee.ID,
			Permission: perm,
		}); err != nil {
			log.Printf("import: failed to grant %s to %s: %v", p.Permission, p.KeyID, err)
		}
	}

	// Import objects
	bucketPath := filepath.Join("buckets", bucketName)
	var importedCount int
	var failedCount int

	for _, obj := range manifest.Objects {
		// Skip folders
		if strings.HasSuffix(obj.Key, "/") {
			if err := objects.CreateFolder(filepath.Join(bucketPath, obj.Key), obj.Key, *b, *user); err != nil {
				log.Printf("import: failed to create folder %s: %v", obj.Key, err)
			}
			continue
		}

		srcPath := filepath.Join(tmpDir, "objects", obj.Key)
		destPath := filepath.Join(bucketPath, obj.Key)

		// Ensure dest directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
			log.Printf("import: failed to create dir for %s: %v", obj.Key, err)
			failedCount++
			continue
		}

		// Copy file from temp to bucket path
		srcFile, err := os.Open(srcPath)
		if err != nil {
			log.Printf("import: missing file for %s: %v", obj.Key, err)
			failedCount++
			continue
		}

		// Determine owner — try to match by key_id, fall back to importing user
		ownerID := user.ID
		if obj.OwnerKeyID != "" {
			if owner, err := auth.LoadAuthorization(obj.OwnerKeyID); err == nil && owner != nil {
				ownerID = owner.ID
			}
		}

		public := obj.Public
		_, err = objects.CreateObject(destPath, obj.Key, *b, srcFile, nil, &types.Authorization{ID: ownerID}, &public)
		srcFile.Close()
		if err != nil {
			log.Printf("import: failed to create object %s: %v", obj.Key, err)
			failedCount++
			continue
		}

		// Object tags
		if len(obj.Tags) > 0 {
			// Get the object we just created to get its ID
			created, err := objects.GetObject(bucketName, obj.Key, nil)
			if err == nil && created != nil {
				tags := make([]types.Tag, len(obj.Tags))
				for i, t := range obj.Tags {
					tags[i] = types.Tag{Key: t.Key, Value: t.Value}
				}
				if err := objects.BulkCreateObjectTags(b.ID, created.ID, tags); err != nil {
					log.Printf("import: failed to create tags for %s: %v", obj.Key, err)
				}
			}
		}

		importedCount++
		if importedCount%1000 == 0 {
			log.Printf("import: imported %d objects...", importedCount)
		}
	}

	log.Printf("import: completed — bucket=%s imported=%d failed=%d skipped_perms=%d",
		bucketName, importedCount, failedCount, len(skippedPerms))

	// Return JSON summary
	result := map[string]any{
		"bucket":              bucketName,
		"objects_imported":    importedCount,
		"objects_failed":      failedCount,
		"permissions_skipped": skippedPerms,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}
