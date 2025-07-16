package types

import (
	"encoding/xml"
	"time"
)

type ObjectContent struct {
	Key          string  `xml:"Key"`
	LastModified IsoTime `xml:"LastModified"`
	ETag         string  `xml:"ETag,omitempty"`
	Size         int64   `xml:"Size"`
}

type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

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
