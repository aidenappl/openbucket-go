//go:build integration

package objects

import (
	"context"
	"testing"
	"time"

	"github.com/aidenappl/openbucket-go/db"
)

// With the pool exhausted, the /_health check (db.Healthy, 2s) fails within
// its budget and an ordinary query fails at QueryTimeout instead of waiting
// forever.
func TestExhaustedPoolFailsFastAndShowsUnhealthy(t *testing.T) {
	seedWalkBucket(t)

	health := func() error { return db.Healthy(2 * time.Second) }
	if err := health(); err != nil {
		t.Fatalf("healthy pool: %v", err)
	}

	db.DB.SetMaxOpenConns(1)
	prev := db.QueryTimeout
	db.QueryTimeout = 1 * time.Second
	t.Cleanup(func() { db.DB.SetMaxOpenConns(100); db.QueryTimeout = prev })

	held, err := db.DB.Conn(context.Background()) // the only connection
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	start := time.Now()
	if health() == nil {
		t.Fatal("exhausted pool reported healthy")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("/_health took %s, want ~2s", d)
	}

	start = time.Now()
	if _, err := ListObjects(walkTestBucket); err == nil {
		t.Fatal("query succeeded with no free connection")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("query took %s to fail, want ~QueryTimeout (1s)", d)
	}

	held.Close()
	if err := health(); err != nil {
		t.Fatalf("after release: %v", err)
	}
}
