package types

import (
	"encoding/xml"
	"time"
)

type Authorization struct {
	ID          int       `db:"id" json:"id"` // DB PK
	Name        string    `xml:"Name" db:"name" json:"name"`
	KeyID       string    `xml:"KEY_ID" db:"key_id" json:"key_id"`
	SecretKey   string    `xml:"SECRET_KEY" db:"secret_key" json:"secret_key"`
	DateCreated time.Time `xml:"Date_Created" db:"date_created" json:"date_created"`
}

type Authorizations struct {
	XMLName        xml.Name        `xml:"Authorizations"`
	Authorizations []Authorization `xml:"Authorization"`
}
