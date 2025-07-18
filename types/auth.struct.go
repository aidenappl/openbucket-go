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
