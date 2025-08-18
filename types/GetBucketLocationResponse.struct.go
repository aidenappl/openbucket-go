package types

import "encoding/xml"

// Works for XML like:
// <LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-west-2</LocationConstraint>
// and for us-east-1 default:
// <LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>
type GetBucketLocationResponse struct {
	XMLName            xml.Name `xml:"LocationConstraint" json:"-"`
	Xmlns              string   `xml:"xmlns,attr,omitempty" json:"-"`
	LocationConstraint *string  `xml:",chardata" json:"LocationConstraint"`
}
