package types

import "encoding/xml"

type GetObjectACLResponse struct {
	XMLName           xml.Name        `xml:"AccessControlPolicy"`
	Xmlns             string          `xml:"xmlns,attr"`
	Owner             UserObject      `json:"owner" xml:"Owner"`
	AccessControlList []MinifiedGrant `xml:"AccessControlList>Grant"`
}
