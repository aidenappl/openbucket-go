package auth

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
)

// DeleteAuthorization removes a credential by ID.
// Returns an error if the credential has active bucket grants.
func DeleteAuthorization(id int) error {
	// Check for active grants
	var count int
	err := db.Psql.
		Select("COUNT(*)").
		From("bucket_permissions").
		Where(sq.Eq{"grantee_id": id}).
		QueryRow().
		Scan(&count)
	if err != nil {
		return fmt.Errorf("check active grants: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("credential has %d active bucket grant(s) — revoke them first", count)
	}

	res, err := db.Psql.
		Delete("authorizations").
		Where(sq.Eq{"id": id}).
		Exec()
	if err != nil {
		return fmt.Errorf("delete authorization: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("credential not found")
	}

	// Evict from cache by scanning all entries
	authCache.Range(func(key, value interface{}) bool {
		entry := value.(*authCacheEntry)
		if entry.auth.ID == id {
			authCache.Delete(key)
			return false
		}
		return true
	})

	return nil
}
