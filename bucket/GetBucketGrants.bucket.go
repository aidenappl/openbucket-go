package bucket

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

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
