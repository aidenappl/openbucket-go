package metadata

import "time"

// XML File Metadata
type Metadata struct {
	ETag   string `xml:"etag" json:"etag"`
	Bucket string `xml:"bucket" json:"bucket"`
	Key    string `xml:"key" json:"key"`
	Tags   struct {
		Tag []string `xml:"tag" json:"tag"` // Tags can be a slice of strings
	} `xml:"tags" json:"tags"`
	VersionId         string    `xml:"versionId" json:"versionId"`
	PreviousVersionId string    `xml:"previousVersionId,omitempty" json:"previousVersionId,omitempty"`
	Owner             string    `xml:"owner" json:"owner"`
	Public            bool      `xml:"public" json:"public"`
	LastModified      time.Time `xml:"lastModified" json:"lastModified"`
	UploadedAt        time.Time `xml:"uploadedAt" json:"uploadedAt"`
}

func New(
	bucket, key string, etag string, public bool, owner string,
) *Metadata {
	return &Metadata{
		ETag:              etag,
		Bucket:            bucket,
		Key:               key,
		Owner:             owner,
		VersionId:         "1", // Default version ID
		PreviousVersionId: "",
		Tags: struct {
			Tag []string `xml:"tag" json:"tag"`
		}{
			Tag: []string{},
		},
		Public:       public,
		LastModified: time.Now(),
		UploadedAt:   time.Now(),
	}
}
