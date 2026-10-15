//go:build integration

package objects

import (
	"sync"
	"testing"
	"time"

	"github.com/aidenappl/openbucket-go/db"
)

// GetObject must never hold one pooled connection while waiting for another.
// With a 2-connection pool and 20 concurrent calls, the old code (tags read
// inside the rows loop) deadlocked: each call held a connection and waited
// for a second. Production hung this way for ~15 hours.
func TestGetObjectUnderPoolPressure(t *testing.T) {
	bucketID := seedWalkBucket(t)
	if _, err := db.DB.Exec(`INSERT INTO object_tags(bucket_id, object_id, tag_key, tag_value)
		SELECT bucket_id, id, 'k', 'v' FROM objects WHERE bucket_id = $1 AND key = 'a.txt'`, bucketID); err != nil {
		t.Fatal(err)
	}

	db.DB.SetMaxOpenConns(2)
	t.Cleanup(func() { db.DB.SetMaxOpenConns(100) })

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			obj, err := GetObject(walkTestBucket, "a.txt", nil)
			if err == nil && (obj == nil || len(obj.Tags.Tag) != 1) {
				t.Errorf("got %+v, want the object with its tag", obj)
			}
			errs <- err
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("GetObject deadlocked under pool pressure")
	}
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
