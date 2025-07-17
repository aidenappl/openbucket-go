package types

import (
	"encoding/xml"
	"time"
)

type Authorization struct {
	Name        string    `xml:"Name"`
	KeyID       string    `xml:"KEY_ID"`
	SecretKey   string    `xml:"SECRET_KEY"`
	DateCreated time.Time `xml:"Date_Created"`
}

type Authorizations struct {
	XMLName        xml.Name        `xml:"Authorizations"`
	Authorizations []Authorization `xml:"Authorization"`
}

type Permissions struct {
	ACL    Permission `xml:"acl"`
	Grants []Grant    `xml:"grants>grant"`
}

type Grant struct {
	KeyID     string     `xml:"keyID"`
	ACL       Permission `xml:"acl"`
	DateAdded time.Time  `xml:"date_added"`
}

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
