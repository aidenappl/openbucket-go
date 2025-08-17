package types

import (
	"encoding/xml"
)

// ObjectMetadata represents the metadata of an object in a bucket.
type ObjectMetadata struct {
	ETag              string     `xml:"ETag" json:"etag"`
	Bucket            string     `xml:"Bucket" json:"bucket"`
	Key               string     `xml:"Key" json:"key"`
	Tags              []Tag      `xml:"Tags>Tag" json:"tags,omitempty"`
	VersionId         string     `xml:"VersionId" json:"versionId"`
	PreviousVersionId string     `xml:"PreviousVersionId,omitempty" json:"previousVersionId,omitempty"`
	Owner             UserObject `xml:"Owner" json:"owner"`
	Public            bool       `xml:"Public" json:"public"`
	Size              int64      `xml:"Size" json:"size"`
	LastModified      IsoTime    `xml:"LastModified" json:"lastModified"`
	UploadedAt        IsoTime    `xml:"UploadedAt" json:"uploadedAt"`
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

// ObjectList represents a list of objects in a bucket.
type ObjectList struct {
	XMLName        xml.Name         `xml:"ListBucketResult"`
	Name           string           `xml:"Name"`
	Prefix         string           `xml:"Prefix"`
	Delimiter      string           `xml:"Delimiter,omitempty"`
	MaxKeys        int              `xml:"MaxKeys"`
	IsTruncated    bool             `xml:"IsTruncated"`
	Contents       []ObjectMetadata `xml:"Contents"`
	CommonPrefixes []CommonPrefix   `xml:"CommonPrefixes,omitempty"`
}
