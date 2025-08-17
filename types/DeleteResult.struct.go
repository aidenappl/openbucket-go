package types

import "encoding/xml"

// DeleteResult is the top-level response for DeleteObjects.
type DeleteResult struct {
	XMLName xml.Name           `xml:"DeleteResult"`
	Xmlns   string             `xml:"xmlns,attr,omitempty"`
	Deleted []DeletedObject    `xml:"Deleted"`
	Errors  []DeleteErrorEntry `xml:"Error"`
}

// DeletedObject represents a successfully deleted object.
type DeletedObject struct {
	Key                   string `xml:"Key"`
	VersionId             string `xml:"VersionId,omitempty"`
	DeleteMarker          bool   `xml:"DeleteMarker,omitempty"`
	DeleteMarkerVersionId string `xml:"DeleteMarkerVersionId,omitempty"`
}

// DeleteErrorEntry represents an error that occurred while deleting an object.
type DeleteErrorEntry struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}
