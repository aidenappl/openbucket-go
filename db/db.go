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
		// No schema prefix — search_path=core in the DSN resolves these
		`DO $$ BEGIN
			CREATE TYPE acl_type AS ENUM (
				'PRIVATE', 'PUBLIC_READ', 'PUBLIC_READ_WRITE',
				'AUTHENTICATED_READ', 'BUCKET_OWNER_READ',
				'BUCKET_OWNER_FULL_CONTROL', 'LOG_DELIVERY_WRITE'
			);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$`,

		`DO $$ BEGIN
			CREATE TYPE permission_type AS ENUM (
				'FULL_CONTROL', 'READ', 'WRITE', 'READ_ACP', 'WRITE_ACP'
			);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$`,

		// Tables
		`CREATE TABLE IF NOT EXISTS authorizations (
			id           serial PRIMARY KEY,
			name         text NOT NULL,
			key_id       varchar(64) NOT NULL UNIQUE,
			secret_key   varchar(128) NOT NULL,
			date_created timestamp DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS buckets (
			id            serial PRIMARY KEY,
			name          text NOT NULL UNIQUE,
			creation_date timestamp DEFAULT now(),
			acl           acl_type DEFAULT 'PRIVATE'::acl_type,
			owner_id      integer REFERENCES authorizations
		)`,

		`CREATE TABLE IF NOT EXISTS bucket_permissions (
			id         serial PRIMARY KEY,
			bucket_id  integer REFERENCES buckets,
			grantee_id integer REFERENCES authorizations,
			permission permission_type NOT NULL,
			date_added timestamp DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS bucket_tags (
			id        serial PRIMARY KEY,
			bucket_id integer REFERENCES buckets,
			tag_key   text NOT NULL,
			tag_value text NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS objects (
			id            serial PRIMARY KEY,
			bucket_id     integer REFERENCES buckets,
			key           text NOT NULL,
			etag          varchar(64) NOT NULL,
			version_id    integer DEFAULT 1,
			owner_id      integer REFERENCES authorizations,
			public        boolean DEFAULT false,
			size          bigint DEFAULT 0,
			last_modified timestamp NOT NULL,
			uploaded_at   timestamp NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS object_tags (
			id        serial PRIMARY KEY,
			bucket_id integer REFERENCES buckets,
			object_id integer REFERENCES objects,
			tag_key   text NOT NULL,
			tag_value text NOT NULL
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_objects_bucket_key ON objects(bucket_id, key)`,
		// Byte order: listings compare and sort keys COLLATE "C" (S3 order), and
		// the delimited walk (objects/WalkObjects) probes this once per entry.
		`CREATE INDEX IF NOT EXISTS idx_objects_bucket_key_c ON objects(bucket_id, key COLLATE "C")`,
		`CREATE INDEX IF NOT EXISTS idx_object_tags_bucket_object ON object_tags(bucket_id, object_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bucket_permissions_bucket_grantee ON bucket_permissions(bucket_id, grantee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bucket_tags_bucket ON bucket_tags(bucket_id)`,
	}

	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if err := ensureUniqueObjectKeys(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("database migrations complete")
	return nil
}

// ensureUniqueObjectKeys keeps one metadata row per (bucket, key) — the most
// recently modified, which describes the file on disk (the last write) — and
// adds the unique index that objects' upsert relies on. A no-op once the index
// exists.
//
// Uploads inserted a new row on every overwrite until October 2026, so
// production buckets held repeated keys. One transaction, with writes to
// objects blocked: a row inserted between the delete and the index build
// would fail the build and stop the server from booting.
func ensureUniqueObjectKeys() error {
	var exists bool
	if err := DB.QueryRow(`SELECT to_regclass('uniq_objects_bucket_key') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// The removed rows and their tags are copied, not discarded:
	// objects_superseded / object_tags_superseded can restore them.
	steps := []string{
		`LOCK TABLE objects IN SHARE ROW EXCLUSIVE MODE`,
		`CREATE TABLE IF NOT EXISTS objects_superseded (LIKE objects)`,
		`CREATE TABLE IF NOT EXISTS object_tags_superseded (LIKE object_tags)`,
		`INSERT INTO objects_superseded
			SELECT o.* FROM objects o
			WHERE EXISTS (
				SELECT 1 FROM objects n
				WHERE n.bucket_id = o.bucket_id AND n.key = o.key
				  AND (n.last_modified, n.id) > (o.last_modified, o.id)
			)`,
		`INSERT INTO object_tags_superseded
			SELECT t.* FROM object_tags t WHERE t.object_id IN (SELECT id FROM objects_superseded)`,
		`DELETE FROM object_tags WHERE object_id IN (SELECT id FROM objects_superseded)`,
		`DELETE FROM objects WHERE id IN (SELECT id FROM objects_superseded)`,
		`CREATE UNIQUE INDEX uniq_objects_bucket_key ON objects(bucket_id, key)`,
	}
	for _, s := range steps {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}

	var removed int
	if err := tx.QueryRow(`SELECT count(*) FROM objects_superseded`).Scan(&removed); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("objects: moved %d superseded metadata rows to objects_superseded; keys are now unique per bucket", removed)
	return nil
}

var DB = func() *sql.DB {
	db, err := sql.Open("postgres", env.Database)
	if err != nil {
		panic(fmt.Errorf("error opening database: %w", err))
	}

	// Configure connection pool — sized for concurrent S3 requests
	// Each middleware chain uses 3-4 DB queries, so we need headroom
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(25)
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
