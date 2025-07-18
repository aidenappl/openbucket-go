package types

import "encoding/xml"

type Bucket struct {
	Name         string     `xml:"Name" json:"name"`
	CreationDate IsoTime    `xml:"CreationDate" json:"creation_date"`
	ACL          Permission `xml:"ACL" json:"acl"`
	Owner        UserObject `xml:"Owner" json:"owner"`
	Grants       []Grant    `xml:"Grants>Grant" json:"grants,omitempty"`
}

type Grant struct {
	Grantee    Grantee    `xml:"Grantee"`
	Permission Permission `xml:"Permission"`
}

type Grantee struct {
	XMLName     xml.Name `xml:"Grantee"`
	ID          string   `xml:"ID,omitempty"`
	DisplayName string   `xml:"DisplayName,omitempty"`
	DateAdded   IsoTime  `xml:"DateAdded"`
}
