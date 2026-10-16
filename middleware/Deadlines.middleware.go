package middleware

import (
	"net/http"
	"time"
)

// Per-request deadlines replace the server-wide ReadTimeout (15 s) and
// WriteTimeout (60 s), which applied to every request alike: any download
// that took longer than a minute — a large file from cdn.trailblaze.to or
// cdn.appleby.cloud on a slow connection — was cut off mid-body, and so was
// any upload slower than 15 s.
//
// Variables only so tests can shorten them.
var (
	DefaultReadDeadline  = 30 * time.Second
	DefaultWriteDeadline = 60 * time.Second
	// TransferDeadline bounds one object upload or download.
	TransferDeadline = 2 * time.Hour
)

// Deadlines sets the default read and write deadlines on every request. It
// wraps the whole router; the server keeps only ReadHeaderTimeout.
func Deadlines(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extendDeadlines(w, DefaultReadDeadline, DefaultWriteDeadline)
		next.ServeHTTP(w, r)
	})
}

// LongTransfer gives an object upload or download route TransferDeadline.
func LongTransfer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		extendDeadlines(w, TransferDeadline, TransferDeadline)
		next(w, r)
	}
}

func extendDeadlines(w http.ResponseWriter, read, write time.Duration) {
	rc := http.NewResponseController(w)
	now := time.Now()
	// ErrNotSupported only outside a real server (tests).
	_ = rc.SetReadDeadline(now.Add(read))
	_ = rc.SetWriteDeadline(now.Add(write))
}
