package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A slow download fails on an ordinary route and completes on LongTransfer.
func TestDeadlines(t *testing.T) {
	prevR, prevW, prevT := DefaultReadDeadline, DefaultWriteDeadline, TransferDeadline
	DefaultReadDeadline, DefaultWriteDeadline, TransferDeadline = 300*time.Millisecond, 300*time.Millisecond, 10*time.Second
	t.Cleanup(func() { DefaultReadDeadline, DefaultWriteDeadline, TransferDeadline = prevR, prevW, prevT })

	slowWrite := func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 8; i++ { // ~800 ms of streaming
			w.Write([]byte(strings.Repeat("x", 64<<10)))
			http.NewResponseController(w).Flush()
			time.Sleep(100 * time.Millisecond)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/short", slowWrite)
	mux.HandleFunc("/long", LongTransfer(slowWrite))
	srv := httptest.NewServer(Deadlines(mux))
	defer srv.Close()

	get := func(path string) (int64, error) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		return io.Copy(io.Discard, resp.Body)
	}
	if n, err := get("/short"); err == nil && n == 8*64<<10 {
		t.Fatal("a slow download on a default route should hit the write deadline")
	}
	if n, err := get("/long"); err != nil || n != 8*64<<10 {
		t.Fatalf("LongTransfer download: %d bytes, %v", n, err)
	}
}
