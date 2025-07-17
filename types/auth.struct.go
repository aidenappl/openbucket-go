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
