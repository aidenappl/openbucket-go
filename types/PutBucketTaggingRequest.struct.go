package types

import "encoding/xml"

type PutBucketTaggingRequest struct {
	XMLName xml.Name    `xml:"Tagging"`
	Xmlns   string      `xml:"xmlns,attr"`
	TagSet  []BucketTag `xml:"TagSet>Tag"`
}
