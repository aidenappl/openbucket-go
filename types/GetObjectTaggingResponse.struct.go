package types

import "encoding/xml"

type GetObjectTaggingResponse struct {
	XMLName xml.Name `xml:"Tagging" json:"-"`
	Xmlns   string   `xml:"xmlns,attr,omitempty" json:"-"`
	TagSet  []Tag    `xml:"TagSet>Tag" json:"TagSet"`
}
