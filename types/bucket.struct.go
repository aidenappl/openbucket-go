package types

import (
	"encoding/xml"
	"time"
)

type Bucket struct {
	ID       int      `db:"id" json:"id"`
	XMLName  xml.Name `xml:"Bucket"`
	Xmlns    string   `xml:"xmlns,attr" json:"-"`
	XmlnsXsi string   `xml:"xmlns:xsi,attr,omitempty" json:"-"`

	Name         string     `xml:"Name" db:"name" json:"name"`
	CreationDate time.Time  `xml:"CreationDate" db:"creation_date" json:"creation_date"`
	ACL          Permission `xml:"ACL" db:"acl" json:"acl"`
	Owner        UserObject `xml:"Owner" json:"owner"`
	Grants       []Grant    `xml:"Grants>Grant" json:"grants,omitempty"`
}

type Grant struct {
	ID         int        `db:"id" json:"id"`
	XMLName    xml.Name   `xml:"Grant"`
	Grantee    Grantee    `xml:"Grantee" json:"grantee"`
	Permission Permission `xml:"Permission" db:"permission" json:"permission"`
	DateAdded  time.Time  `xml:"DateAdded" db:"date_added" json:"date_added,omitempty"`
}

type Grantee struct {
	XMLName     xml.Name `xml:"Grantee"`
	Type        string   `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr,omitempty" json:"type,omitempty"`
	ID          string   `xml:"ID,omitempty" db:"id" json:"id,omitempty"`
	DisplayName string   `xml:"DisplayName,omitempty" db:"display_name" json:"display_name,omitempty"`
	URI         string   `xml:"URI,omitempty" db:"uri" json:"uri,omitempty"`
}
