package bucket

import (
	"log"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

type ListBucketsParams struct {
	Prefix  *string
	Filter  *ListBucketsParamFilter
	Include *ListBucketsParamInclude
}

type ListBucketsParamFilter struct {
	OwnerKeyID *string
	OwnerID    *int
}

type ListBucketsParamInclude struct {
	Grants bool
}

func ListBuckets(params *ListBucketsParams) (*[]types.Bucket, error) {
	// Build the query
	query := db.Psql.Select(
		"buckets.id",
		"buckets.name",
		"buckets.creation_date",
		"buckets.acl",

		"authorizations.key_id",
		"authorizations.name",
	).From("buckets").
		LeftJoin("authorizations ON buckets.owner_id = authorizations.id")

	// Add prefix filter
	if params.Prefix != nil {
		query = query.Where(sq.Like{"buckets.name": *params.Prefix + "%"})
	}

	// Check for filters
	if params.Filter != nil {
		if params.Filter.OwnerID != nil {
			query = query.Where(sq.Eq{"buckets.owner_id": *params.Filter.OwnerID})
		}
	}

	// Run the query
	rows, err := query.Query()
	if err != nil {
		log.Println("Error querying buckets:", err)
		return nil, err
	}
	defer rows.Close()

	var buckets []types.Bucket
	for rows.Next() {
		var bucket types.Bucket
		if err := rows.Scan(
			&bucket.ID,
			&bucket.Name,
			&bucket.CreationDate,
			&bucket.ACL,

			&bucket.Owner.ID,
			&bucket.Owner.DisplayName,
		); err != nil {
			log.Println("Error scanning bucket:", err)
			return nil, err
		}

		if params.Include != nil && params.Include.Grants {
			grants, err := GetBucketGrants(bucket.Name)
			if err != nil {
				log.Println("Error getting bucket grants:", err)
				return nil, err
			}
			bucket.Grants = grants
		}

		buckets = append(buckets, bucket)
	}

	return &buckets, nil
}

func ListBucketsXML(params *ListBucketsParams) (*types.BucketList, error) {
	buckets, err := ListBuckets(params)
	if err != nil {
		log.Println("Error listing buckets:", err)
		return nil, err
	}

	bucketList := &types.BucketList{
		Buckets: struct {
			Bucket []types.Bucket `xml:"Bucket"`
		}{
			Bucket: make([]types.Bucket, len(*buckets)),
		},
	}

	copy(bucketList.Buckets.Bucket, *buckets)

	return bucketList, nil
}
