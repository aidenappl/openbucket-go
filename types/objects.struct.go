package types

import (
	"encoding/xml"
	"time"
)

// ObjectList defines the structure of the XML response for listing objects in a bucket.
// types.ObjectContent
type ObjectContent struct {
	Key          string  `xml:"Key"`
	LastModified IsoTime `xml:"LastModified"`
	ETag         string  `xml:"ETag,omitempty"`
	Size         int64   `xml:"Size"`
}

// types.CommonPrefix
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

// types.ObjectList
type ObjectList struct {
	XMLName        xml.Name        `xml:"ListBucketResult"`
	Name           string          `xml:"Name"`
	Prefix         string          `xml:"Prefix"`
	Delimiter      string          `xml:"Delimiter,omitempty"`
	MaxKeys        int             `xml:"MaxKeys"`
	IsTruncated    bool            `xml:"IsTruncated"`
	Contents       []ObjectContent `xml:"Contents"`
	CommonPrefixes []CommonPrefix  `xml:"CommonPrefixes,omitempty"`
}

type IsoTime time.Time

func (t IsoTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	v := time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z")
	return e.EncodeElement(v, start)
}
