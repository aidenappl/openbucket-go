package types

import (
	"encoding/xml"
)

type TagSet struct {
	XMLName xml.Name `xml:"Tags"`
	Tag     []Tag    `xml:"Tag"`
}

type ETag string

func (e ETag) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	return enc.EncodeElement(`"`+string(e)+`"`, start) // S3 quotes ETag
}

// ObjectMetadata represents the metadata of an object in a bucket.
type ObjectMetadata struct {
	Xmlns string `xml:"xmlns,attr,omitempty"` // set to "http://s3.amazonaws.com/doc/2006-03-01/" if you want the S3 ns

	ETag   ETag   `xml:"ETag" json:"etag"`
	Bucket string `xml:"Bucket" json:"bucket"`
	Key    string `xml:"Key" json:"key"`

	Tags TagSet `xml:"Tags" json:"tags"`

	VersionId         string `xml:"VersionId" json:"versionId"`
	PreviousVersionId string `xml:"PreviousVersionId,omitempty" json:"previousVersionId,omitempty"`

	Owner        UserObject `xml:"Owner" json:"owner"`
	Public       bool       `xml:"Public" json:"public"`
	Size         int64      `xml:"Size" json:"size"`
	LastModified IsoTime    `xml:"LastModified" json:"lastModified"`
	UploadedAt   IsoTime    `xml:"UploadedAt" json:"uploadedAt"`
}

type Tag struct {
	Key   string `xml:"Key" json:"key"`
	Value string `xml:"Value" json:"value"`
}

// OwnerObject represents the owner of an object in the bucket.
type UserObject struct {
	ID          string `xml:"ID" json:"id"`
	DisplayName string `xml:"DisplayName" json:"displayName"`
}

// CommonPrefix represents a common prefix in the object listing.
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
	Size   int64  `xml:"Size,omitempty"`
}

type ObjectList struct {
	XMLName               xml.Name         `xml:"ListBucketResult"`
	Xmlns                 string           `xml:"xmlns,attr,omitempty"` // set to S3 ns
	IsTruncated           bool             `xml:"IsTruncated"`
	Contents              []ObjectMetadata `xml:"Contents"`
	Name                  string           `xml:"Name"`
	Prefix                string           `xml:"Prefix"`
	Delimiter             string           `xml:"Delimiter,omitempty"`
	MaxKeys               int              `xml:"MaxKeys"`
	CommonPrefixes        []CommonPrefix   `xml:"CommonPrefixes,omitempty"`
	EncodingType          string           `xml:"EncodingType,omitempty"`      // "url" if requested
	KeyCount              int              `xml:"KeyCount"`                    // number of keys in this page
	ContinuationToken     string           `xml:"ContinuationToken,omitempty"` // echo input if you want
	NextContinuationToken string           `xml:"NextContinuationToken,omitempty"`
	StartAfter            string           `xml:"StartAfter,omitempty"`
}
