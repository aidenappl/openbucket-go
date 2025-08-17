package types

type CopyObjectResult struct {
	ETag         string  `json:"ETag"`
	LastModified IsoTime `json:"LastModified"`
}
