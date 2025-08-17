package types

import "encoding/xml"

// DeleteRequest models the <Delete> request body for DeleteObjects.
type DeleteRequest struct {
	XMLName xml.Name           `xml:"Delete"`
	Xmlns   string             `xml:"xmlns,attr,omitempty"`
	Quiet   bool               `xml:"Quiet,omitempty"`
	Objects []ObjectIdentifier `xml:"Object"`
}

// ObjectIdentifier models each <Object> entry.
type ObjectIdentifier struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId,omitempty"`
}
