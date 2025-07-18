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
	XMLName    xml.Name   `xml:"Grant"`
	XmlnsXsi   string     `xml:"xmlns:xsi,attr,omitempty"`
	Grantee    Grantee    `xml:"Grantee"`
	Permission Permission `xml:"Permission"`
	DateAdded  IsoTime    `xml:"DateAdded" json:"date_added,omitempty"`
}

type Grantee struct {
	XMLName     xml.Name `xml:"Grantee"`
	Type        string   `xml:"xsi:type,attr"`
	ID          string   `xml:"ID,omitempty"`
	DisplayName string   `xml:"DisplayName,omitempty"`
	URI         string   `xml:"URI,omitempty"`
}
