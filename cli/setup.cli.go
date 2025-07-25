package cli

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func SetupCLI() {
	var rootCmd = &cobra.Command{Use: "openbucket"}

	// `openbucket create-bucket [bucket_name]`
	// This command creates a new bucket with the specified name.
	var createCmd = &cobra.Command{
		Use:   "create-bucket [bucket_name]",
		Short: "Create a new bucket",
		Args:  cobra.MinimumNArgs(1),
		Run:   createBucket,
	}
	rootCmd.AddCommand(createCmd)

	// `openbucket generate-credentials`
	// This command generates new credentials.
	var credentialsCmd = &cobra.Command{
		Use:   "generate-credentials",
		Short: "Generate new credentials",
		Run:   generateCredentials,
	}
	rootCmd.AddCommand(credentialsCmd)

	// `openbucket grant [bucket_name] [key_id] [?acl]`
	// This command grants access to a bucket for a user identified by key_id.
	var grantCmd = &cobra.Command{
		Use:   "grant [bucket_name] [key_id] [acl]",
		Short: "Grant access to a bucket for a user",
		Args:  cobra.RangeArgs(2, 3),
		Run:   grant,
	}
	rootCmd.AddCommand(grantCmd)

	// `openbucket grant-update [bucket_name] [key_id] [acl]`
	// This command grants access to a bucket for a user identified by key_id.
	var grantUpdateCmd = &cobra.Command{
		Use:   "grant-update [bucket_name] [key_id] [acl]",
		Short: "Update access to a bucket for a user",
		Args:  cobra.ExactArgs(3),
		Run:   grantUpdate,
	}
	rootCmd.AddCommand(grantUpdateCmd)

	// `openbucket list-buckets`
	// This command lists all buckets or the buckets accessible by a specific user.
	var listBucketsCmd = &cobra.Command{
		Use:   "list-buckets",
		Short: "List all buckets",
		Run:   listBuckets,
	}
	rootCmd.AddCommand(listBucketsCmd)

	// `openbucket permissions [bucket_name]`
	// This command shows the permissions for a specific bucket.
	var permissionsCmd = &cobra.Command{
		Use:   "permissions [bucket_name]",
		Short: "Show permissions for a bucket",
		Args:  cobra.ExactArgs(1),
		Run:   permissions,
	}
	rootCmd.AddCommand(permissionsCmd)

	// `openbucket list-objects [bucket_name]`
	// This command lists objects in a specified bucket, with optional flags for prefix and delimiter.
	var listObjectsCmd = &cobra.Command{
		Use:   "list-objects [bucket]",
		Short: "List objects (and folders) in a bucket",
		Long:  `List objects in a bucket. Optional flags --prefix and --delimiter work like AWS S3.`,
		Args:  cobra.ExactArgs(1),

		RunE: listObjects,
	}
	listObjectsCmd.Flags().StringP("prefix", "p", "", "only keys that begin with this prefix")
	listObjectsCmd.Flags().StringP("delimiter", "d", "/", "path delimiter (default '/')")

	rootCmd.AddCommand(listObjectsCmd)

	// `openbucket list-credentials`
	// This command lists all credentials.
	var listCredentialsCmd = &cobra.Command{
		Use:   "list-credentials",
		Short: "List all credentials",
		Run:   listCredentials,
	}
	rootCmd.AddCommand(listCredentialsCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Println("Error executing CLI command:", err)
		os.Exit(1)
	}
}
