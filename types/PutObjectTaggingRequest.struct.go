package types

import "encoding/xml"

type PutObjectTaggingRequest struct {
	XMLName xml.Name `xml:"Tagging"`
	Xmlns   string   `xml:"xmlns,attr"`
	TagSet  []Tag    `xml:"TagSet>Tag"`
}
