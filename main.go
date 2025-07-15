package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/routers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

// Server Code
func startServer() {
	// Create a new router using Gorilla Mux
	r := mux.NewRouter()

	// Handle the PUT request for uploading files
	r.HandleFunc("/{bucket}/{key:.*}", routers.HandleUpload).Methods(http.MethodPut)
	// Handle the GET request for listing buckets
	r.HandleFunc("/", routers.HandleListBuckets).Methods(http.MethodGet)
	// Handle the GET request for listing objects in a bucket
	r.HandleFunc("/{bucket}", routers.HandleListObjects).Methods(http.MethodGet)
	// Handle head of an object
	r.HandleFunc("/{bucket}/{key:.*}", routers.HandleHeadObject).Methods(http.MethodHead)
	// Handle the GET request for downloading an object
	r.HandleFunc("/{bucket}/{key:.*}", routers.HandleDownload).Methods(http.MethodGet)
	// Handle delete of an object
	r.HandleFunc("/{bucket}/{key:.*}", routers.HandleDelete).Methods(http.MethodDelete)
	// Create Bucket
	r.HandleFunc("/{bucket}", routers.HandleCreateBucket).Methods(http.MethodPut)
	// Delete Bucket
	r.HandleFunc("/{bucket}", routers.HandleDeleteBucket).Methods(http.MethodDelete)

	// Start the server
	fmt.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

// CLI Code
func createBucket(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Bucket name is required")
		return
	}

	bucketName := args[0]
	fmt.Printf("Creating bucket: %s\n", bucketName)

	err := handler.CreateBucket(bucketName)
	if err != nil {
		fmt.Println("Error creating bucket:", err)
		return
	}

	fmt.Println("Bucket created successfully")
}

func setupCLI() {
	var rootCmd = &cobra.Command{Use: "openbucket"}

	var cliCmd = &cobra.Command{
		Use:   "cli",
		Short: "OpenBucket CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("OpenBucket CLI is running. Use 'openbucket create-bucket [bucket_name]' to create a bucket.")
		},
	}
	rootCmd.AddCommand(cliCmd)
	var createCmd = &cobra.Command{
		Use:   "create-bucket [bucket_name]",
		Short: "Create a new bucket",
		Args:  cobra.MinimumNArgs(1),
		Run:   createBucket,
	}

	// Add the create-bucket command to the root command
	rootCmd.AddCommand(createCmd)

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error executing CLI command:", err)
		os.Exit(1)
	}
}

// main function
func main() {
	// Check if the first argument is 'cli', if so run CLI commands
	if len(os.Args) > 1 {
		// Run the CLI logic
		setupCLI()
	} else {
		// Otherwise, run the server
		startServer()
	}
}
