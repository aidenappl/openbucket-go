package db

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/env"
	_ "github.com/lib/pq"
)

const (
	DefaultListLimit = 50
	MaximumListLimit = 100

	ErNoReferencedRow     = 1215
	ErDupEntry            = 1062
	ErDupEntryWithKeyName = 1586
)

func PingDB() error {
	db, err := sql.Open("postgres", env.Database)
	if err != nil {
		return err
	}

	ping := db.Ping()
	db.Close()

	return ping
}

var DB = func() *sql.DB {
	db, err := sql.Open("postgres", env.Database)
	if err != nil {
		panic(fmt.Errorf("error opening database: %w", err))
	}

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
