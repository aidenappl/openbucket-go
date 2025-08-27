package types

import (
	"encoding/xml"
)

type TagSet struct {
	XMLName xml.Name `xml:"Tags"`
	Tag     []Tag    `xml:"Tag"`
}

// ObjectMetadata represents the metadata of an object in a bucket.
type ObjectMetadata struct {
	Xmlns string `xml:"xmlns,attr,omitempty"` // set to "http://s3.amazonaws.com/doc/2006-03-01/" if you want the S3 ns

	ETag   string `xml:"ETag" json:"etag"`
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
	XMLName        xml.Name         `xml:"ListBucketResult"`
	Xmlns          string           `xml:"xmlns,attr,omitempty"` // set same ns if desired
	Name           string           `xml:"Name"`
	Prefix         string           `xml:"Prefix"`
	Delimiter      string           `xml:"Delimiter,omitempty"`
	MaxKeys        int              `xml:"MaxKeys"`
	IsTruncated    bool             `xml:"IsTruncated"`
	EncodingType   string           `xml:"EncodingType,omitempty"`
	Contents       []ObjectMetadata `xml:"Contents"`
	CommonPrefixes []CommonPrefix   `xml:"CommonPrefixes,omitempty"`
}
