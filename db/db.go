package db

import (
	"database/sql"
	"fmt"
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
