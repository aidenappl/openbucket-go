package types

type CopyObjectResult struct {
	ETag         ETag    `json:"ETag"`
	LastModified IsoTime `json:"LastModified"`
}
