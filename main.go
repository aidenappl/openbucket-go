package main

import (
	"log"
	"net/http"
	"os"

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

	// Create a new router
	r := mux.NewRouter()

	// Middleware for handling request state (Host & Request ID)
	r.Use(middleware.RequestState)

	// Logging middleware for console output
	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/", middleware.HalfAuthorized(routers.HandleListBuckets)).Methods(http.MethodGet)

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

	// Start the server
	log.Println("✅ Server started at http://localhost:" + env.Port)
	err := http.ListenAndServe(":"+env.Port, r)
	if err != nil {
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
