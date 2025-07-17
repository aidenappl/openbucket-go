package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aidenappl/openbucket-go/auth"
	"github.com/aidenappl/openbucket-go/handler"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

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

func listBuckets(cmd *cobra.Command, args []string) {
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
}

func generateCredentials(cmd *cobra.Command, args []string) {

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
}

func grant(cmd *cobra.Command, args []string) {
	bucketName := args[0]
	keyID := args[1]
	err := handler.GrantAccess(bucketName, keyID)
	if err != nil {
		fmt.Println("Error granting access:", err)
		return
	}
	fmt.Printf("Access granted to %s for bucket %s\n", keyID, bucketName)
}

func permissions(cmd *cobra.Command, args []string) {
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
	table0.Header([]string{"Bucket ACL"})
	table0.Append([]string{fmt.Sprintf("%s", permissions.ACL)})
	table0.Render()

	table1 := tablewriter.NewWriter(os.Stdout)
	table1.Header([]string{"Key ID", "ACL", "Date Added"})
	table1.Bulk(permissions.Grants)
	table1.Render()
}

func listObjects(cmd *cobra.Command, args []string) error {

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
}

func listCredentials(cmd *cobra.Command, args []string) {
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
}
