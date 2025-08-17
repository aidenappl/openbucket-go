package types

import "encoding/xml"

type ListAllMyBucketsResult struct {
	XMLName xml.Name       `xml:"ListAllMyBucketsResult"`
	Xmlns   string         `xml:"xmlns,attr"`
	Owner   UserObject     `xml:"Owner"`
	Buckets ListAllBuckets `xml:"Buckets"`
}

type ListAllBuckets struct {
	Bucket []ListAllBucket `xml:"Bucket"`
}

type ListAllBucket struct {
	Name         string  `xml:"Name"`
	CreationDate IsoTime `xml:"CreationDate"` // ISO8601/RFC3339 e.g. 2025-08-17T15:40:21.138Z
}
