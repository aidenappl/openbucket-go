package types

import "encoding/xml"

// GetBucketTaggingResponse represents:
// <Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
//
//	<TagSet>
//	  <Tag><Key>...</Key><Value>...</Value></Tag>
//	</TagSet>
//
// </Tagging>
type GetBucketTaggingResponse struct {
	XMLName xml.Name `xml:"Tagging" json:"-"`
	Xmlns   string   `xml:"xmlns,attr,omitempty" json:"-"`
	TagSet  []Tag    `xml:"TagSet>Tag" json:"TagSet"`
}
