package routers

import (
	"log"
	"net/http"
	"time"

	"github.com/aidenappl/openbucket-go/db"
)

// HandleHealth serves GET /_health: 200 "ok" when a pooled database connection
// is usable within 2s, else 503. See main.go for why it goes through the pool.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-store")
	if err := db.Healthy(2 * time.Second); err != nil {
		log.Println("health: database unavailable:", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("database unavailable\n"))
		return
	}
	w.Write([]byte("ok\n"))
}
