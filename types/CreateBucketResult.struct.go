package types

import "encoding/xml"

type CreateBucketResult struct {
	XMLName  xml.Name `xml:"CreateBucketResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Location string   `xml:"Location"`
}
