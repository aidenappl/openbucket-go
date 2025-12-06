package types

import (
	"database/sql/driver"
	"encoding/xml"
)

//
// ========== TAGGING ==========
//

type Tag struct {
	ID       int    `db:"id" json:"id"`
	ObjectID int    `db:"object_id" json:"object_id"`
	Key      string `xml:"Key" db:"tag_key" json:"key"`
	Value    string `xml:"Value" db:"tag_value" json:"value"`
}

type TagSet struct {
	XMLName xml.Name `xml:"Tags"`
	Tag     []Tag    `xml:"Tag"`
}

//
// ========== ETAG ==========
//

type ETag string

func (e ETag) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	// AWS S3 wraps ETag values in quotes
	return enc.EncodeElement(`"`+string(e)+`"`, start)
}

func (e ETag) ToString() string {
	return string(e)
}

func (e ETag) Value() (driver.Value, error) { // for DB insert
	return string(e), nil
}

//
// ========== OBJECT METADATA ==========
//

type ObjectMetadata struct {
	ID        int        `db:"id" json:"id"`
	BucketID  int        `db:"bucket_id" json:"bucket_id"`
	Key       string     `xml:"Key" db:"key" json:"key"`
	ETag      *ETag      `xml:"ETag" db:"etag" json:"etag"`
	VersionID int        `xml:"VersionId" db:"version_id" json:"version_id"`
	OwnerID   int        `db:"owner_id" json:"owner_id"`
	Owner     UserObject `xml:"Owner" json:"owner"`

	Public       bool    `xml:"Public" db:"public" json:"public"`
	Size         int64   `xml:"Size" db:"size" json:"size"`
	LastModified IsoTime `xml:"LastModified" db:"last_modified" json:"lastModified"`
	UploadedAt   IsoTime `xml:"UploadedAt" db:"uploaded_at" json:"uploadedAt"`

	Tags TagSet `xml:"Tags" json:"tags"`
}

//
// ========== OWNER ==========
//

type UserObject struct {
	ID          string `xml:"ID" db:"id" json:"id"`
	DisplayName string `xml:"DisplayName" db:"display_name" json:"displayName"`
}

//
// ========== OBJECT LISTING ==========
//

type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
	Size   int64  `xml:"Size,omitempty"`
}

type ObjectList struct {
	XMLName               xml.Name         `xml:"ListBucketResult"`
	Xmlns                 string           `xml:"xmlns,attr,omitempty"`
	IsTruncated           bool             `xml:"IsTruncated"`
	Contents              []ObjectMetadata `xml:"Contents"`
	Name                  string           `xml:"Name"`
	Prefix                string           `xml:"Prefix"`
	Public                bool             `xml:"Public,omitempty"`
	Delimiter             string           `xml:"Delimiter,omitempty"`
	MaxKeys               int              `xml:"MaxKeys"`
	CommonPrefixes        []CommonPrefix   `xml:"CommonPrefixes,omitempty"`
	EncodingType          string           `xml:"EncodingType,omitempty"`
	KeyCount              int              `xml:"KeyCount"`
	ContinuationToken     string           `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string           `xml:"NextContinuationToken,omitempty"`
	StartAfter            string           `xml:"StartAfter,omitempty"`
}
