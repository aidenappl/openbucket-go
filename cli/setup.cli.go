package cli

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func SetupCLI() {
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
		Run:   generateCredentials,
	}
	rootCmd.AddCommand(credentialsCmd)

	var grantCmd = &cobra.Command{
		Use:   "grant [bucket_name] [key_id]",
		Short: "Grant access to a bucket for a user",
		Args:  cobra.ExactArgs(2),
		Run:   grant,
	}
	rootCmd.AddCommand(grantCmd)

	var listBucketsCmd = &cobra.Command{
		Use:   "list-buckets",
		Short: "List all buckets",
		Run:   listBuckets,
	}
	rootCmd.AddCommand(listBucketsCmd)

	var permissionsCmd = &cobra.Command{
		Use:   "permissions [bucket_name]",
		Short: "Show permissions for a bucket",
		Args:  cobra.ExactArgs(1),
		Run:   permissions,
	}
	rootCmd.AddCommand(permissionsCmd)

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
