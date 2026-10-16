package db

import (
	"context"
	"database/sql"
	"time"
)

// QueryTimeout bounds every query made through Psql or Timed: getting a pooled
// connection, running the statement, and reading its rows.
//
// ⚠️ database/sql waits for a free connection forever. When the pool was
// exhausted on 2026-10-15 every request waited forever too, and
// cdn.trailblaze.to hung for ~15 hours (openbucket-go#33). With a deadline an
// exhausted pool fails requests, which then release their goroutines, instead
// of hanging the server.
var QueryTimeout = 30 * time.Second

// timedRunner runs queries on DB under QueryTimeout. It deliberately has the
// plain (non-Context) method set, so squirrel calls these methods.
type timedRunner struct{ db *sql.DB }

// Timed is DB with QueryTimeout applied; use it for raw SQL outside Psql.
var Timed = timedRunner{db: DB}

func (t timedRunner) Exec(query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	return t.db.ExecContext(ctx, query, args...)
}

// Query's context must outlive this call (the caller reads the rows), so it
// is released by the deadline rather than on return.
func (t timedRunner) Query(query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		cancel()
		return nil, err
	}
	time.AfterFunc(QueryTimeout, cancel)
	return rows, nil
}

func (t timedRunner) QueryRow(query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	time.AfterFunc(QueryTimeout, cancel)
	return t.db.QueryRowContext(ctx, query, args...)
}

// Healthy reports whether a pooled connection can be had and used within d.
// It goes through the pool on purpose: an exhausted pool is the failure the
// /_health endpoint exists to catch.
//
// The deadline is enforced here, not only through the context: when the
// database itself stops responding, lib/pq's cancellation opens a new
// connection to it and blocks too, so PingContext alone overran d (10s+ with
// the DB container paused). The ping goroutine finishes whenever the database
// does.
func Healthy(d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	done := make(chan error, 1)
	go func() {
		defer cancel()
		done <- DB.PingContext(ctx)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		return context.DeadlineExceeded
	}
}
