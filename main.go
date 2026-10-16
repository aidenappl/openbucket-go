package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aidenappl/openbucket-go/cli"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/routers"
	"github.com/gorilla/mux"
)

func startServer() {

	// Ping database
	if err := db.PingDB(); err != nil {
		log.Fatal("Error pinging database:", err)
	} else {
		log.Println("✅ Database connection established")
	}

	// Run auto-migrations (idempotent — safe on every startup)
	if err := db.RunMigrations(); err != nil {
		log.Fatal("Error running migrations:", err)
	} else {
		log.Println("✅ Database schema ready")
	}

	// Create a new router
	r := mux.NewRouter()

	// Middleware for handling request state (Host & Request ID)
	r.Use(middleware.RequestState)

	// Logging middleware for console output
	r.Use(middleware.LoggingMiddleware)

	// ── Admin API (bearer token auth, JSON responses) ────────────────────
	admin := r.PathPrefix("/admin/").Subrouter()
	admin.Use(middleware.AdminAuth)

	// Credentials
	admin.HandleFunc("/credentials", routers.HandleAdminListCredentials).Methods(http.MethodGet)
	admin.HandleFunc("/credentials", routers.HandleAdminCreateCredential).Methods(http.MethodPost)
	admin.HandleFunc("/credentials/{id}", routers.HandleAdminDeleteCredential).Methods(http.MethodDelete)

	// Buckets
	admin.HandleFunc("/buckets", routers.HandleAdminListBuckets).Methods(http.MethodGet)
	admin.HandleFunc("/buckets", routers.HandleAdminCreateBucket).Methods(http.MethodPost)
	admin.HandleFunc("/buckets/{bucket}", routers.HandleAdminGetBucket).Methods(http.MethodGet)
	admin.HandleFunc("/buckets/{bucket}", routers.HandleAdminDeleteBucket).Methods(http.MethodDelete)
	admin.HandleFunc("/buckets/{bucket}/stats", routers.HandleAdminGetBucketStats).Methods(http.MethodGet)
	admin.HandleFunc("/buckets/{bucket}/acl", routers.HandleAdminUpdateBucketACL).Methods(http.MethodPut)

	// Grants
	admin.HandleFunc("/buckets/{bucket}/grants", routers.HandleAdminListGrants).Methods(http.MethodGet)
	admin.HandleFunc("/buckets/{bucket}/grants", routers.HandleAdminCreateGrant).Methods(http.MethodPost)
	admin.HandleFunc("/buckets/{bucket}/grants/{keyId}", routers.HandleAdminUpdateGrant).Methods(http.MethodPut)
	admin.HandleFunc("/buckets/{bucket}/grants/{id}", routers.HandleAdminDeleteGrant).Methods(http.MethodDelete)

	// ── Health (unauthenticated) ─────────────────────────────────────────
	// 200 only if a pooled DB connection is usable within 2s, so a deadlocked
	// or exhausted pool shows as unhealthy — a plain "process is up" check
	// passed throughout the 2026-10-15 outage (openbucket-go#33). Probed by
	// openbucket-api's storage prober. Registered before /{bucket}: "_health"
	// can't be a bucket name.
	r.HandleFunc("/_health", routers.HandleHealth).Methods(http.MethodGet, http.MethodHead)

	// ── S3-Compatible API (AWS SigV4 auth, XML responses) ────────────────
	r.HandleFunc("/", middleware.HalfAuthorized(routers.HandleListBuckets)).Methods(http.MethodGet)
	r.HandleFunc("/_import", middleware.HalfAuthorized(routers.HandleImportBucket)).Methods(http.MethodPost)

	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleBucket)).Methods(http.MethodGet)
	r.HandleFunc("/{bucket}", middleware.HalfAuthorized(routers.HandleCreateBucket)).Methods(http.MethodPut)
	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleModifyBucket)).Methods(http.MethodPost)
	r.HandleFunc("/{bucket}", middleware.HalfAuthorized(routers.HandleBucketHead)).Methods(http.MethodHead)
	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleDeleteBucket)).Methods(http.MethodDelete)

	// Object downloads and uploads get long deadlines (middleware.LongTransfer).
	r.HandleFunc("/{bucket}/{key:.*}", middleware.LongTransfer(middleware.Authorized(routers.HandleDownload))).Methods(http.MethodGet)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleHeadObject)).Methods(http.MethodHead)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleDelete)).Methods(http.MethodDelete)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.LongTransfer(middleware.Authorized(routers.HandleUpload))).Methods(http.MethodPut)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleUploadModification)).Methods(http.MethodPost)

	// Read/write deadlines are per request (middleware.Deadlines; object
	// transfers extended by LongTransfer). Server-wide ReadTimeout/WriteTimeout
	// cut every download off at 60 s and every upload at 15 s.
	server := &http.Server{
		Addr:              ":" + env.Port,
		Handler:           middleware.Deadlines(r),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Println("✅ Server started at http://localhost:" + env.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func healthcheck() int {
	c := http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + env.Port + "/_health")
	if err != nil {
		log.Println("healthcheck:", err)
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

func main() {
	// Check if sending cli commands or starting server
	// `openbucket healthcheck`: exit 0 when this server's /_health is OK.
	// For a container HEALTHCHECK (the image has no curl/wget).
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

	if len(os.Args) > 1 {
		// Run CLI handler
		cli.SetupCLI()
	} else {
		// Start the server
		startServer()
	}
}
