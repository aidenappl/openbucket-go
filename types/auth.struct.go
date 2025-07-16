package types

import (
	"encoding/xml"
	"time"
)

// Authorization represents each user in the global authorizations file
type Authorization struct {
	Name        string    `xml:"Name"`
	KeyID       string    `xml:"KEY_ID"`
	SecretKey   string    `xml:"SECRET_KEY"`
	DateCreated time.Time `xml:"Date_Created"`
}

// Authorizations represents the structure of the authorizations XML
type Authorizations struct {
	XMLName        xml.Name        `xml:"Authorizations"`
	Authorizations []Authorization `xml:"Authorization"`
}

// Permissions represents the structure of the bucket-specific permissions file
type Permissions struct {
	AllowGlobalRead  bool    `xml:"global_read"`
	AllowGlobalWrite bool    `xml:"global_write"`
	Grants           []Grant `xml:"grants>grant"`
}

type Grant struct {
	KeyID     string    `xml:"keyID"`
	DateAdded time.Time `xml:"date_added"`
}

// Metadata represents the structure of the metadata XML file.
type Metadata struct {
	ETag         string    `xml:"etag"`
	Bucket       string    `xml:"bucket"`
	Key          string    `xml:"key"`
	Tags         string    `xml:"tags"`
	VersionID    string    `xml:"versionId"`
	Owner        string    `xml:"owner"`
	Public       bool      `xml:"public"`
	LastModified time.Time `xml:"lastModified"`
	UploadedAt   time.Time `xml:"uploadedAt"`
}
