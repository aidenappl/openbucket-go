package types

import "encoding/xml"

const XsiNS = "http://www.w3.org/2001/XMLSchema-instance"

// Root element returned by GET /bucket?acl
type AccessControlPolicy struct {
	XMLName           xml.Name   `xml:"AccessControlPolicy"`
	XmlnsXsi          string     `xml:"xmlns:xsi,attr,omitempty"` // add automatically
	Owner             UserObject `xml:"Owner"`
	AccessControlList []Grant    `xml:"AccessControlList>Grant"`
}
