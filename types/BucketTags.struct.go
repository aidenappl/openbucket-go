package types

import "encoding/xml"

type BucketTagging struct {
	XMLName xml.Name `xml:"Tagging"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	TagSet  []Tag    `xml:"TagSet>Tag"`
}
