package types

import "encoding/xml"

type Bucket struct {
	XMLName xml.Name `xml:"Bucket"`
	// Default namespace for all child elements
	Xmlns string `xml:"xmlns,attr"` // set to "http://s3.amazonaws.com/doc/2006-03-01/"
	// Declare the xsi namespace once at the root
	XmlnsXsi string `xml:"xmlns:xsi,attr,omitempty"` // set to "http://www.w3.org/2001/XMLSchema-instance"

	Name         string     `xml:"Name" json:"name"`
	CreationDate IsoTime    `xml:"CreationDate" json:"creation_date"`
	ACL          Permission `xml:"ACL" json:"acl"`
	Owner        UserObject `xml:"Owner" json:"owner"`
	Grants       []Grant    `xml:"Grants>Grant" json:"grants,omitempty"`
}

type Grant struct {
	XMLName    xml.Name   `xml:"Grant"`
	Grantee    Grantee    `xml:"Grantee"`
	Permission Permission `xml:"Permission"`
	DateAdded  IsoTime    `xml:"DateAdded" json:"date_added,omitempty"`
}

type Grantee struct {
	XMLName xml.Name `xml:"Grantee"`
	// Use the namespace URI + local name for namespaced attributes
	// This makes Go emit xsi:type, using the in-scope xmlns:xsi from the root.
	Type        string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr,omitempty"`
	ID          string `xml:"ID,omitempty"`
	DisplayName string `xml:"DisplayName,omitempty"`
	URI         string `xml:"URI,omitempty"`
}
