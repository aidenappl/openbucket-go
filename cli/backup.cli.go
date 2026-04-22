package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/bucket"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/objects"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
	"github.com/spf13/cobra"
)

func exportBucket(cmd *cobra.Command, args []string) error {
	bucketName := args[0]
	outputPath := args[1]

	if err := db.PingDB(); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Load bucket with grants
	b, err := bucket.GetBucket(bucketName)
	if err != nil || b == nil {
		return fmt.Errorf("bucket %q not found", bucketName)
	}

	fmt.Printf("Exporting bucket: %s\n", bucketName)

	// Load bucket tags
	tags, err := bucket.GetBucketTags(b.ID)
	if err != nil {
		return fmt.Errorf("failed to load bucket tags: %w", err)
	}

	// List all objects
	objs, err := objects.ListObjects(bucketName, &objects.ListObjectsOptions{BucketID: b.ID})
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	fmt.Printf("  Objects: %d\n", len(objs))

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

	if tags != nil {
		for _, t := range *tags {
			manifest.Bucket.Tags = append(manifest.Bucket.Tags, types.BackupTag{
				Key: t.Key, Value: t.Value,
			})
		}
	}

	for _, g := range b.Grants {
		manifest.Permissions = append(manifest.Permissions, types.BackupPermission{
			KeyID:      g.Grantee.ID,
			Permission: string(g.Permission),
		})
	}

	for _, obj := range objs {
		meta := types.BackupObjectMetadata{
			Key:          obj.Key,
			ETag:         string(*obj.ETag),
			Size:         obj.Size,
			Public:       obj.Public,
			LastModified: time.Time(obj.LastModified),
			OwnerKeyID:   obj.Owner.ID,
		}
		objTags, err := objects.GetObjectTags(b.ID, obj.ID)
		if err == nil {
			for _, t := range objTags {
				meta.Tags = append(meta.Tags, types.BackupTag{Key: t.Key, Value: t.Value})
			}
		}
		manifest.Objects = append(manifest.Objects, meta)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write manifest
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: "manifest.json", Size: int64(len(manifestBytes)),
		Mode: 0644, ModTime: time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to write manifest header: %w", err)
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Write object files
	bucketPath := util.GetBucketPath(bucketName)
	var written int
	for _, obj := range objs {
		filePath := filepath.Join(bucketPath, obj.Key)
		fi, err := os.Stat(filePath)
		if err != nil || fi.IsDir() {
			continue
		}

		if err := tw.WriteHeader(&tar.Header{
			Name: filepath.Join("objects", obj.Key), Size: fi.Size(),
			Mode: 0644, ModTime: fi.ModTime(),
		}); err != nil {
			log.Printf("  warning: skipping %s: %v", obj.Key, err)
			continue
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("  warning: skipping %s: %v", obj.Key, err)
			continue
		}
		if _, err := io.Copy(tw, file); err != nil {
			file.Close()
			return fmt.Errorf("failed writing %s: %w", obj.Key, err)
		}
		file.Close()
		written++

		if written%1000 == 0 {
			fmt.Printf("  Archived %d/%d objects...\n", written, len(objs))
		}
	}

	fmt.Printf("  Exported %d objects to %s\n", written, outputPath)
	return nil
}

func importBucket(cmd *cobra.Command, args []string) error {
	archivePath := args[0]

	if err := db.PingDB(); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	if err := db.RunMigrations(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	nameOverride, _ := cmd.Flags().GetString("name")

	// Open archive
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("not a valid gzip file: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// Extract to temp dir
	tmpDir, err := os.MkdirTemp("", "openbucket-import-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var manifest types.BackupManifest
	var foundManifest bool

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid tar archive: %w", err)
		}

		clean := filepath.Clean(header.Name)
		if strings.Contains(clean, "..") {
			return fmt.Errorf("archive contains path traversal")
		}

		if header.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("failed to read manifest: %w", err)
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("invalid manifest: %w", err)
			}
			foundManifest = true
			continue
		}

		if strings.HasPrefix(header.Name, "objects/") && header.Typeflag == tar.TypeReg {
			destPath := filepath.Join(tmpDir, header.Name)
			if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
				continue
			}
			out, err := os.Create(destPath)
			if err != nil {
				continue
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				continue
			}
			out.Close()
		}
	}

	if !foundManifest {
		return fmt.Errorf("archive does not contain manifest.json")
	}
	if manifest.Version != 1 {
		return fmt.Errorf("unsupported manifest version: %d", manifest.Version)
	}

	bucketName := manifest.Bucket.Name
	if nameOverride != "" {
		bucketName = nameOverride
	}

	fmt.Printf("Importing bucket: %s (%d objects)\n", bucketName, len(manifest.Objects))

	// Check if bucket exists
	existing, _ := bucket.GetBucket(bucketName)
	if existing != nil {
		return fmt.Errorf("bucket %q already exists — use --name to import under a different name", bucketName)
	}

	// We need an owner. Use the first authorization in the system, or create if none exist.
	auths, err := auth.LoadAuthorizations()
	if err != nil || auths == nil || len(auths.Authorizations) == 0 {
		return fmt.Errorf("no credentials exist — run 'openbucket generate-credentials' first")
	}
	owner := auths.Authorizations[0]

	if err := bucket.CreateBucket(bucketName, owner.ID); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	b, err := bucket.GetBucket(bucketName)
	if err != nil || b == nil {
		return fmt.Errorf("failed to fetch created bucket")
	}

	// Set ACL
	acl := types.ConvertToPermission(manifest.Bucket.ACL)
	if acl != types.ACLUnknown {
		db.Psql.Update("buckets").
			Set("acl", string(acl)).
			Where("id = ?", b.ID).
			Exec()
	}

	// Bucket tags
	for _, t := range manifest.Bucket.Tags {
		bucket.CreateBucketTag(b.ID, types.BucketTag{Key: t.Key, Value: t.Value})
	}

	// Permissions
	for _, p := range manifest.Permissions {
		if p.KeyID == owner.KeyID {
			continue
		}
		grantee, err := auth.LoadAuthorization(p.KeyID)
		if err != nil || grantee == nil {
			fmt.Printf("  Skipping permission for unknown key_id: %s\n", p.KeyID)
			continue
		}
		perm := types.ConvertToPermission(p.Permission)
		if perm == types.ACLUnknown {
			perm = types.READ
		}
		bucket.NewGrant(bucket.NewGrantReq{
			BucketID: b.ID, GranteeID: grantee.ID, Permission: perm,
		})
	}

	// Import objects
	bucketPath := filepath.Join("buckets", bucketName)
	var imported, failed int

	for _, obj := range manifest.Objects {
		if strings.HasSuffix(obj.Key, "/") {
			objects.CreateFolder(filepath.Join(bucketPath, obj.Key), obj.Key, *b, owner)
			continue
		}

		srcPath := filepath.Join(tmpDir, "objects", obj.Key)
		destPath := filepath.Join(bucketPath, obj.Key)

		if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
			failed++
			continue
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			log.Printf("  missing file: %s", obj.Key)
			failed++
			continue
		}

		ownerID := owner.ID
		if obj.OwnerKeyID != "" {
			if o, err := auth.LoadAuthorization(obj.OwnerKeyID); err == nil && o != nil {
				ownerID = o.ID
			}
		}

		public := obj.Public
		_, err = objects.CreateObject(destPath, obj.Key, *b, srcFile, nil, &types.Authorization{ID: ownerID}, &public)
		srcFile.Close()
		if err != nil {
			log.Printf("  failed: %s: %v", obj.Key, err)
			failed++
			continue
		}

		// Object tags
		if len(obj.Tags) > 0 {
			created, err := objects.GetObject(bucketName, obj.Key, nil)
			if err == nil && created != nil {
				tags := make([]types.Tag, len(obj.Tags))
				for i, t := range obj.Tags {
					tags[i] = types.Tag{Key: t.Key, Value: t.Value}
				}
				objects.BulkCreateObjectTags(b.ID, created.ID, tags)
			}
		}

		imported++
		if imported%1000 == 0 {
			fmt.Printf("  Imported %d/%d objects...\n", imported, len(manifest.Objects))
		}
	}

	fmt.Printf("  Import complete: %d imported, %d failed\n", imported, failed)
	return nil
}
