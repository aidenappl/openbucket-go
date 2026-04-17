package auth

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

// authCacheEntry holds a cached authorization with expiry time
type authCacheEntry struct {
	auth      *types.Authorization
	expiresAt time.Time
}

// authCache is a simple in-memory cache for authorization lookups
var authCache sync.Map

// authCacheTTL is how long cached auth entries are valid
const authCacheTTL = 5 * time.Minute

// getCachedAuth retrieves an auth from cache if valid
func getCachedAuth(keyID string) (*types.Authorization, bool) {
	if entry, ok := authCache.Load(keyID); ok {
		cached := entry.(*authCacheEntry)
		if time.Now().Before(cached.expiresAt) {
			return cached.auth, true
		}
		// Expired - delete from cache
		authCache.Delete(keyID)
	}
	return nil, false
}

// setCachedAuth stores an auth in cache
func setCachedAuth(keyID string, auth *types.Authorization) {
	authCache.Store(keyID, &authCacheEntry{
		auth:      auth,
		expiresAt: time.Now().Add(authCacheTTL),
	})
}

func SaveCredentials(creds *types.Authorization) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}
	if creds.KeyID == "" || creds.SecretKey == "" {
		return fmt.Errorf("credentials KeyID and SecretKey cannot be empty")
	}
	if creds.Name == "" {
		return fmt.Errorf("credentials Name cannot be empty")
	}
	if creds.DateCreated.IsZero() {
		creds.DateCreated = time.Now()
	}

	// Insert using Squirrel
	query := db.Psql.
		Insert("authorizations").
		Columns("name", "key_id", "secret_key", "date_created").
		Values(creds.Name, creds.KeyID, creds.SecretKey, creds.DateCreated).
		Suffix("RETURNING id")

	// Run the query and fetch the new ID
	var id int
	err := query.QueryRow().Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to insert credentials: %w", err)
	}
	creds.ID = id

	log.Printf("✅ Credentials saved for %s (id=%d)\n", creds.Name, creds.ID)
	return nil
}

func LoadAuthorizations() (*types.Authorizations, error) {
	rows, err := db.Psql.
		Select("id", "name", "key_id", "secret_key", "date_created").
		From("authorizations").
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query authorizations: %v", err)
	}
	defer rows.Close()

	var authorizations types.Authorizations
	for rows.Next() {
		var auth types.Authorization
		if err := rows.Scan(&auth.ID, &auth.Name, &auth.KeyID, &auth.SecretKey, &auth.DateCreated); err != nil {
			return nil, fmt.Errorf("failed to scan authorization: %v", err)
		}
		authorizations.Authorizations = append(authorizations.Authorizations, auth)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over authorizations: %v", err)
	}

	return &authorizations, nil
}

func LoadAuthorization(keyID string) (*types.Authorization, error) {
	rows, err := db.Psql.
		Select("id", "name", "key_id", "secret_key", "date_created").
		From("authorizations").
		Where(sq.Eq{"key_id": keyID}).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query authorizations: %v", err)
	}
	defer rows.Close()

	var auth types.Authorization
	for rows.Next() {
		if err := rows.Scan(&auth.ID, &auth.Name, &auth.KeyID, &auth.SecretKey, &auth.DateCreated); err != nil {
			return nil, fmt.Errorf("failed to scan authorization: %v", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over authorizations: %v", err)
	}

	if auth.ID == 0 {
		return nil, nil
	}

	return &auth, nil
}

func CheckUserExists(keyID string) (*types.Authorization, error) {
	// Check cache first
	if cached, ok := getCachedAuth(keyID); ok {
		return cached, nil
	}

	row := db.Psql.
		Select("id", "name", "key_id", "secret_key", "date_created").
		From("authorizations").
		Where(sq.Eq{"key_id": keyID}).
		QueryRow()

	var a types.Authorization
	err := row.Scan(&a.ID, &a.Name, &a.KeyID, &a.SecretKey, &a.DateCreated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	// Store in cache
	setCachedAuth(keyID, &a)

	return &a, nil
}

func CheckUserPermissions(keyID, bucketName string) (*types.Grant, error) {
	row := db.Psql.
		Select(
			"bp.permission",
			"bp.date_added",
			"a.key_id",
			"a.name",
		).
		From("bucket_permissions bp").
		Join("buckets b ON bp.bucket_id = b.id").
		Join("authorizations a ON bp.grantee_id = a.id").
		Where(sq.And{
			sq.Eq{"a.key_id": keyID},
			sq.Eq{"b.name": bucketName},
		}).
		QueryRow()

	var grant types.Grant
	err := row.Scan(&grant.Permission, &grant.DateAdded, &grant.Grantee.ID, &grant.Grantee.DisplayName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}
	return &grant, nil
}
