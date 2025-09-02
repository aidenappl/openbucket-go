package types

import "encoding/xml"

type GetBucketTaggingResponse struct {
	XMLName xml.Name    `xml:"Tagging" json:"-"`
	Xmlns   string      `xml:"xmlns,attr,omitempty" json:"-"`
	TagSet  []BucketTag `xml:"TagSet>Tag" json:"TagSet"`
}
