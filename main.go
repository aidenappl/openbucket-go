package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/routers"
	"github.com/gorilla/mux"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func startServer() {

	r := mux.NewRouter()

	r.Use(middleware.RequestState)

	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleUpload)).Methods(http.MethodPut)

	r.HandleFunc("/", middleware.Authorized(routers.HandleListBuckets)).Methods(http.MethodGet)

	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleListObjects)).Methods(http.MethodGet)

	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleHeadObject)).Methods(http.MethodHead)

	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleDownload)).Methods(http.MethodGet)

	r.HandleFunc("/{bucket}/{key:.*}", middleware.Authorized(routers.HandleDelete)).Methods(http.MethodDelete)

	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleCreateBucket)).Methods(http.MethodPut)

	r.HandleFunc("/{bucket}", middleware.Authorized(routers.HandleDeleteBucket)).Methods(http.MethodDelete)

	log.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func createBucket(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Println("Bucket name is required")
		return
	}

	bucketName := args[0]
	fmt.Printf("Creating bucket: %s\n", bucketName)

	err := handler.CreateBucket(bucketName)
	if err != nil {
		log.Println("Error creating bucket:", err)
		return
	}

	log.Println("Bucket created successfully")
}

func setupCLI() {
	var rootCmd = &cobra.Command{Use: "openbucket"}

	var createCmd = &cobra.Command{
		Use:   "create-bucket [bucket_name]",
		Short: "Create a new bucket",
		Args:  cobra.MinimumNArgs(1),
		Run:   createBucket,
	}
	rootCmd.AddCommand(createCmd)

	var credentialsCmd = &cobra.Command{
		Use:   "generate-credentials",
		Short: "Generate new credentials",
		Run: func(cmd *cobra.Command, args []string) {

			fmt.Print("Enter credential name: ")
			name := ""
			fmt.Scanln(&name)
			if name == "" {
				fmt.Println("Credential name cannot be empty")
				return
			}
			creds := handler.GenerateCredentials()
			creds.Name = name
			handler.SaveCredentials(creds)
			fmt.Printf("Generated Credentials:\nAccess Key ID: %s\nSecret Access Key: %s\n", creds.KeyID, creds.SecretKey)
		},
	}
	rootCmd.AddCommand(credentialsCmd)

	var grantCmd = &cobra.Command{
		Use:   "grant [bucket_name] [key_id]",
		Short: "Grant access to a bucket for a user",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bucketName := args[0]
			keyID := args[1]
			err := handler.GrantAccess(bucketName, keyID)
			if err != nil {
				fmt.Println("Error granting access:", err)
				return
			}
			fmt.Printf("Access granted to %s for bucket %s\n", keyID, bucketName)
		},
	}
	rootCmd.AddCommand(grantCmd)

	var listBucketsCmd = &cobra.Command{
		Use:   "list-buckets",
		Short: "List all buckets",
		Run: func(cmd *cobra.Command, args []string) {
			buckets, err := handler.ListBuckets()
			if err != nil {
				fmt.Println("Error listing buckets:", err)
				return
			}
			if len(*buckets) == 0 {
				fmt.Println("No buckets found")
				return
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"Bucket Name", "Creation Date"})
			table.Bulk(*buckets)
			table.Render()
		},
	}
	rootCmd.AddCommand(listBucketsCmd)

	var permissionsCmd = &cobra.Command{
		Use:   "permissions [bucket_name]",
		Short: "Show permissions for a bucket",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucketName := args[0]
			permissions, err := auth.LoadPermissions(bucketName)
			if err != nil {
				fmt.Println("Error getting bucket permissions:", err)
				return
			}
			if len(permissions.Grants) == 0 {
				fmt.Printf("No permissions set for bucket %s\n", bucketName)
				return
			}

			table0 := tablewriter.NewWriter(os.Stdout)
			table0.Header([]string{"Global Read", "Global Write"})
			table0.Append([]string{fmt.Sprintf("%t", permissions.AllowGlobalRead), fmt.Sprintf("%t", permissions.AllowGlobalWrite)})
			table0.Render()

			table1 := tablewriter.NewWriter(os.Stdout)
			table1.Header([]string{"Key ID", "Date Added"})
			table1.Bulk(permissions.Grants)
			table1.Render()
		},
	}
	rootCmd.AddCommand(permissionsCmd)

	var listObjectsCmd = &cobra.Command{
		Use:   "list-objects [bucket]",
		Short: "List objects (and folders) in a bucket",
		Long:  `List objects in a bucket. Optional flags --prefix and --delimiter work like AWS S3.`,
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {

			bucket := args[0]

			prefix, _ := cmd.Flags().GetString("prefix")
			delimiter, _ := cmd.Flags().GetString("delimiter")
			if delimiter == "" {
				delimiter = "/"
			}

			objs, err := handler.ListObjects(bucket)
			if err != nil {
				return fmt.Errorf("list objects: %w", err)
			}
			if len(objs) == 0 {
				fmt.Printf("No objects found in bucket %q (prefix %q)\n", bucket, prefix)
				return nil
			}

			tbl := tablewriter.NewWriter(os.Stdout)
			tbl.Header([]string{"Key", "Type", "Size", "Last Modified"})

			for _, o := range objs {
				typ := "FILE"
				size := fmt.Sprintf("%d", o.Size)

				if strings.HasSuffix(o.Key, "/") {
					typ = "DIR"
					size = "-"
				}
				tbl.Append([]string{
					o.Key,
					typ,
					size,
					"",
				})
			}
			tbl.Render()
			return nil
		},
	}
	listObjectsCmd.Flags().StringP("prefix", "p", "", "only keys that begin with this prefix")
	listObjectsCmd.Flags().StringP("delimiter", "d", "/", "path delimiter (default '/')")

	rootCmd.AddCommand(listObjectsCmd)

	var listCredentialsCmd = &cobra.Command{
		Use:   "list-credentials",
		Short: "List all credentials",
		Run: func(cmd *cobra.Command, args []string) {
			credentials, err := auth.LoadAuthorizations()
			if err != nil {
				fmt.Println("Error loading credentials:", err)
				return
			}
			if len(credentials.Authorizations) == 0 {
				fmt.Println("No credentials found")
				return
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"Key ID", "Secret Key", "Name", "Created At"})
			for _, cred := range credentials.Authorizations {
				table.Append([]string{cred.KeyID, cred.SecretKey, cred.Name, cred.DateCreated.Format("2006-01-02 15:04:05")})
			}
			table.Render()
		},
	}
	rootCmd.AddCommand(listCredentialsCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Println("Error executing CLI command:", err)
		os.Exit(1)
	}
}

func main() {

	if len(os.Args) > 1 {

		setupCLI()
	} else {

		startServer()
	}
}
