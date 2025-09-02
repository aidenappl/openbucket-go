package types

import "encoding/xml"

const XsiNS = "http://www.w3.org/2001/XMLSchema-instance"
const XsiSchemaLocation = "http://www.w3.org/2001/XMLSchema-instance"
const XsiTypeCanonicalUser = "CanonicalUser"
const XsiTypeBucket = "Bucket"
const XsiTypeObject = "Object"
const XsiNS_Default = "http://s3.amazonaws.com/doc/2006-03-01/"

// Root element returned by GET /bucket?acl
type AccessControlPolicyResponse struct {
	XMLName           xml.Name   `xml:"AccessControlPolicy"`
	XmlnsXsi          string     `xml:"xmlns:xsi,attr,omitempty"` // add automatically
	Owner             UserObject `xml:"Owner"`
	AccessControlList []Grant    `xml:"AccessControlList>Grant"`
}
