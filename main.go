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

	// ── S3-Compatible API (AWS SigV4 auth, XML responses) ────────────────
	r.HandleFunc("/", middleware.HalfAuthorized(routers.HandleListBuckets)).Methods(http.MethodGet)
	r.HandleFunc("/_import", middleware.HalfAuthorized(routers.HandleImportBucket)).Methods(http.MethodPost)

	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleBucket)).Methods(http.MethodGet)
	r.HandleFunc("/{bucket}", middleware.HalfAuthorized(routers.HandleCreateBucket)).Methods(http.MethodPut)
	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleModifyBucket)).Methods(http.MethodPost)
	r.HandleFunc("/{bucket}", middleware.HalfAuthorized(routers.HandleBucketHead)).Methods(http.MethodHead)
	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleDeleteBucket)).Methods(http.MethodDelete)

	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleDownload)).Methods(http.MethodGet)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleHeadObject)).Methods(http.MethodHead)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleDelete)).Methods(http.MethodDelete)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleUpload)).Methods(http.MethodPut)
	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleUploadModification)).Methods(http.MethodPost)

	// Start the server with timeouts to prevent goroutine/connection accumulation
	server := &http.Server{
		Addr:         ":" + env.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("✅ Server started at http://localhost:" + env.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func main() {
	// Check if sending cli commands or starting server
	if len(os.Args) > 1 {
		// Run CLI handler
		cli.SetupCLI()
	} else {
		// Start the server
		startServer()
	}
}
