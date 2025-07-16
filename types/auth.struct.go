package types

import "encoding/xml"

// Authorization represents each user in the global authorizations file
type Authorization struct {
	Name        string `xml:"Name"`
	KeyID       string `xml:"KEY_ID"`
	SecretKey   string `xml:"SECRET_KEY"`
	DateCreated string `xml:"Date_Created"`
}

// Authorizations represents the structure of the authorizations XML
type Authorizations struct {
	XMLName       xml.Name        `xml:"Authorizations"`
	Authorization []Authorization `xml:"Authorization"`
}

// Permissions represents the structure of the bucket-specific permissions file
type Permissions struct {
	AllowGlobalRead  bool     `xml:"global_read"`
	AllowGlobalWrite bool     `xml:"global_write"`
	Grants           []string `xml:"grants>grant"`
}

// Metadata represents the structure of the metadata XML file.
type Metadata struct {
	ETag         string `xml:"etag"`
	Bucket       string `xml:"bucket"`
	Key          string `xml:"key"`
	Tags         string `xml:"tags"`
	VersionID    string `xml:"versionId"`
	Owner        string `xml:"owner"`
	Public       bool   `xml:"public"`
	LastModified string `xml:"lastModified"`
	UploadedAt   string `xml:"uploadedAt"`
}
