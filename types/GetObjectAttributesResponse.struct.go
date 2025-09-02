package types

import "encoding/xml"

type GetObjectAttributesResponse struct {
	XMLName xml.Name `xml:"GetObjectAttributesResponse"`
	Xmlns   string   `xml:"xmlns,attr"`

	ETag       string `xml:"ETag,omitempty"`
	ObjectSize int64  `xml:"ObjectSize,omitempty"`

	StorageClass string    `xml:"StorageClass,omitempty"`
	Checksum     *Checksum `xml:"Checksum,omitempty"`
	// TODO: add ObjectParts, ObjectLockMode, etc.
}

type Checksum struct {
	CRC32  string `xml:"ChecksumCRC32,omitempty"`
	SHA1   string `xml:"ChecksumSHA1,omitempty"`
	SHA256 string `xml:"ChecksumSHA256,omitempty"`
	CRC32C string `xml:"ChecksumCRC32C,omitempty"`
}
