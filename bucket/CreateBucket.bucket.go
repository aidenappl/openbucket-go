package bucket

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
	"github.com/aidenappl/openbucket-go/util"
)

func GetBucket(bucketName string) (*types.Bucket, error) {
	rows, err := db.Psql.Select(
		"buckets.id",
		"buckets.name",
		"buckets.creation_date",
		"buckets.acl",

		"authorizations.id",
		"authorizations.name",
	).From("buckets").
		LeftJoin("authorizations ON buckets.owner_id = authorizations.id").
		Where(sq.Or{
			sq.Eq{"buckets.name": bucketName},
		}).
		Query()
	if err != nil {
		return nil, fmt.Errorf("query buckets: %w", err)
	}
	defer rows.Close()

	var bucket types.Bucket
	if rows.Next() {
		if err := rows.Scan(
			&bucket.ID,
			&bucket.Name,
			&bucket.CreationDate,
			&bucket.ACL,

			&bucket.Owner.ID,
			&bucket.Owner.DisplayName,
		); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}

		// Get the bucket grants
		grants, err := GetBucketGrants(bucketName)
		if err != nil {
			log.Println("Error getting bucket grants:", err)
			return nil, fmt.Errorf("error getting bucket grants: %w", err)
		}

		bucket.Grants = grants

		return &bucket, nil
	}
	return nil, nil

}

func GetBucketGrants(bucketName string) ([]types.Grant, error) {
	rows, err := db.Psql.Select(
		"bucket_permissions.id",
		"bucket_permissions.permission",
		"bucket_permissions.date_added",

		"authorizations.name",
		"authorizations.key_id",
	).From("bucket_permissions").
		LeftJoin("buckets ON bucket_permissions.bucket_id = buckets.id").
		LeftJoin("authorizations ON bucket_permissions.grantee_id = authorizations.id").
		Where(sq.Eq{"buckets.name": bucketName}).
		Query()
	if err != nil {
		return nil, fmt.Errorf("query bucket grants: %w", err)
	}
	defer rows.Close()

	var grants []types.Grant
	for rows.Next() {
		var grant types.Grant
		if err := rows.Scan(
			&grant.ID,
			&grant.Permission,
			&grant.DateAdded,

			&grant.Grantee.DisplayName,
			&grant.Grantee.ID,
		); err != nil {
			return nil, fmt.Errorf("scan bucket grant: %w", err)
		}
		grants = append(grants, grant)
	}

	return grants, nil
}

func CreateBucket(bucketName string, owner int) error {
	// Does it already exist?
	bucket, err := GetBucket(bucketName)
	if err != nil {
		log.Println("Error getting bucket:", err)
		return fmt.Errorf("error getting bucket: %w", err)
	}

	if bucket != nil {
		log.Println("Bucket already exists:", bucketName)
		return fmt.Errorf("bucket already exists: %s", bucketName)
	}

	// Create bucket
	query := db.Psql.Insert("buckets").Columns(
		"name",
		"owner_id",
		"creation_date",
		"acl",
	).Values(
		bucketName,
		owner,
		time.Now(),
		types.BUCKET_ACLPrivate,
	).Suffix("RETURNING id")

	// Execute the query
	var bucketID int
	err = query.QueryRow().Scan(&bucketID)
	if err != nil {
		return fmt.Errorf("create bucket %s: %w", bucketName, err)
	}

	// Grant owner full permissions
	err = auth.SaveNewGrant(auth.SaveNewGrantReq{
		BucketID:   bucketID,
		GranteeID:  owner,
		Permission: types.FULL_CONTROL,
	})
	if err != nil {
		return fmt.Errorf("failed to save new grant: %v", err)
	}

	// Create local file
	if !util.BucketExists(bucketName) {
		// Create the local directory for the bucket
		err = CreateBucketDirectory(bucketName)
		if err != nil {
			return fmt.Errorf("failed to create local bucket directory: %v", err)
		}
	}

	return nil
}

func CreateBucketDirectory(bucketName string) error {
	// Create the local directory for the bucket
	err := os.MkdirAll(filepath.Join("buckets", bucketName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create local bucket directory: %v", err)
	}
	return nil
}
