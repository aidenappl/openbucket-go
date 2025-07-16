package types

import (
	"encoding/xml"
	"time"
)

// ObjectList defines the structure of the XML response for listing objects in a bucket.
type ObjectList struct {
	XMLName  xml.Name        `xml:"ListBucketResult"`
	Contents []ObjectContent `xml:"Contents"`
}

type ObjectContent struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	CreatedAt    time.Time `xml:"CreatedAt"`
	ETag         string    `xml:"ETag"`
	Size         int64     `xml:"Size"`
}
