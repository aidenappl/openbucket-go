package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/env"
	_ "github.com/lib/pq"
)

const (
	DefaultListLimit = 50
	MaximumListLimit = 100
)

func PingDB() error {
	return DB.Ping()
}

// RunMigrations ensures the core schema, ENUM types, tables, and indexes exist.
// Safe to run on every startup — all statements use IF NOT EXISTS.
func RunMigrations() error {
	migrations := []string{
		// Schema
		`CREATE SCHEMA IF NOT EXISTS core`,

		// ENUM types (wrapped in DO blocks for idempotency)
		`DO $$ BEGIN
			CREATE TYPE core.acl_type AS ENUM (
				'PRIVATE', 'PUBLIC_READ', 'PUBLIC_READ_WRITE',
				'AUTHENTICATED_READ', 'BUCKET_OWNER_READ',
				'BUCKET_OWNER_FULL_CONTROL', 'LOG_DELIVERY_WRITE'
			);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$`,

		`DO $$ BEGIN
			CREATE TYPE core.permission_type AS ENUM (
				'FULL_CONTROL', 'READ', 'WRITE', 'READ_ACP', 'WRITE_ACP'
			);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$`,

		// Tables
		`CREATE TABLE IF NOT EXISTS core.authorizations (
			id           serial PRIMARY KEY,
			name         text NOT NULL,
			key_id       varchar(64) NOT NULL UNIQUE,
			secret_key   varchar(128) NOT NULL,
			date_created timestamp DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS core.buckets (
			id            serial PRIMARY KEY,
			name          text NOT NULL UNIQUE,
			creation_date timestamp DEFAULT now(),
			acl           core.acl_type DEFAULT 'PRIVATE',
			owner_id      integer REFERENCES core.authorizations
		)`,

		`CREATE TABLE IF NOT EXISTS core.bucket_permissions (
			id         serial PRIMARY KEY,
			bucket_id  integer REFERENCES core.buckets,
			grantee_id integer REFERENCES core.authorizations,
			permission core.permission_type NOT NULL,
			date_added timestamp DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS core.bucket_tags (
			id        serial PRIMARY KEY,
			bucket_id integer REFERENCES core.buckets,
			tag_key   text NOT NULL,
			tag_value text NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS core.objects (
			id            serial PRIMARY KEY,
			bucket_id     integer REFERENCES core.buckets,
			key           text NOT NULL,
			etag          varchar(64) NOT NULL,
			version_id    integer DEFAULT 1,
			owner_id      integer REFERENCES core.authorizations,
			public        boolean DEFAULT false,
			size          bigint DEFAULT 0,
			last_modified timestamp NOT NULL,
			uploaded_at   timestamp NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS core.object_tags (
			id        serial PRIMARY KEY,
			bucket_id integer REFERENCES core.buckets,
			object_id integer REFERENCES core.objects,
			tag_key   text NOT NULL,
			tag_value text NOT NULL
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_objects_bucket_key ON core.objects(bucket_id, key)`,
		`CREATE INDEX IF NOT EXISTS idx_object_tags_bucket_object ON core.object_tags(bucket_id, object_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bucket_permissions_bucket_grantee ON core.bucket_permissions(bucket_id, grantee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bucket_tags_bucket ON core.bucket_tags(bucket_id)`,
	}

	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	log.Println("database migrations complete")
	return nil
}

var DB = func() *sql.DB {
	db, err := sql.Open("postgres", env.Database)
	if err != nil {
		panic(fmt.Errorf("error opening database: %w", err))
	}

	// Configure connection pool for performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db
}()

var Psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(DB)

var ErrNoRows = sql.ErrNoRows

type Queryable interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
