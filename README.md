# OpenBucket Server

**OpenBucket Server** (openbucket-go) is an open-source, S3-compatible object storage API designed for local development, testing, and education. It provides an AWS S3-style interface so you can use the familiar aws s3api and SDKs against your own local service.

### Associated Tools (in-development)

- [OpenBucket Web](https://github.com/aidenappl/openbucket-web): Web interface to interact with OpenBucket
- [OpenBucket API](https://github.com/aidenappl/openbucket-api): Backend handler for OpenBucket Web

---

## Features

- **S3 API compatibility**
  Supports core S3 operations such as bucket creation, object upload/download, ACLs, tagging, and versioning.

- **PostgreSQL backend**
  Stores metadata (buckets, objects, ACLs, tags, permissions) in a structured relational schema for reliability and queryability.

- **Extensible design**
  Built in Go, with modular handlers and database helpers so additional S3 APIs can be implemented incrementally.

- **XML wire protocol**
  Fully compliant with S3's XML-over-HTTP responses, ensuring compatibility with AWS CLI and SDKs (which parse XML into JSON automatically).

---

## Why OpenBucket

OpenBucket is an extension of one of my "1-Week Projects". 

My goal with the short-term projects I select is to either learn a new language, skill, toolset or challenge myself technically.

OpenBucket started as a small-scale GUI for S3 but branched into a curiosity of the inner-workings of many AWS services. 

Now and moving forward I want OpenBucket to:

- Be easily self-hosted whether within a container or on a desktop device.
- Be navigable on a users hosted operating system.
- Open more doors into building other AWS services.

---

## Getting Started

### Requirements

- Go 1.21+
- PostgreSQL 14+
- Docker (optional, for local DB setup)

### Clone & Build

```bash
git clone https://github.com/aidenappl/openbucket-go
cd openbucket
go build -o openbucket .
```

### Database Setup

Create a Postgres database and schema, find sql build script [here](https://github.com/aidenappl/openbucket-go/db/OpenBucket_V1.sql) or in the db folder.

Set your connection string (with schema path) in your environment variables.

```
export DATABASE="postgres://username:password@localhost:5432/openbucket?sslmode=disable&search_path=core"
```

### Run the sever

```
./openbucket
```

---

## Using with AWS CLI

In order the use with the AWS CLI you must first generate credentials after setting up openbucket.

```
go run . generate-credentials
```

Follow the prompts and you will be generated an AccessKey and SecretKey. From there initialize the aws cli with those keys

```
aws configure
```

**IMPORTANT** as OpenBucket requires you to set "openbucket" as your region within your AWS configuration. This can be changed in your openbucket server environment file.

---

## Design Notes

- Folders: Just like AWS S3, OpenBucket doesn't store folders separately; prefixes in object keys (path/to/file) are parsed into "CommonPrefixes" at listing time.

- ACLs & Permissions: Modeled after S3's AccessControlPolicy, with enums for bucket ACLs and object permissions.

- Tags: Stored separately for buckets and objects (bucket_tags, object_tags).

---

## Pipeline

- [ ] Hashing keys stored in DB
- [ ] Versioning support
- [ ] Consolidating & cleaning code
- [ ] Improving speeds
- [ ] Completing OpenBucket CLI
- [x] 100% coverage of CORE commands
- [ ] 100% coverage of SECONDARY commands
- [ ] 100% coverage of TERTIARY commands

---

## S3Api Coverage

Below you can see the coverage of version 1 and planned support for future releases. "Command" is the aws s3api command, "Method" is the http method that hits the openbucket server, "Handler" is which routing handler is hit when that command is called, "Version" describes which version was that command implemented or will that command be implemented in. Built and Funct are for internal tracking as the aws s3api is a massive command set.

### Coverage Breakdown

- **CORE commands**: 100% Coverage (Version 1)
- **SECONDARY commands**: 0% Coverage (Version 2)
- **TERTIARY commands**: 0% Coverage (Version 3+)

### Detailed Command Coverage

| Command                                         | METHOD | HANDLER                  | VERSION | BUILT | FUNCT |
| ----------------------------------------------- | ------ | ------------------------ | ------- | ----- | ----- |
| copy-object                                     | PUT    | HandleUpload             | 1       | TRUE  | TRUE  |
| create-bucket                                   | PUT    | HandleCreateBucket       | 1       | TRUE  | TRUE  |
| delete-bucket                                   | DELETE | HandleDeleteBucket       | 1       | TRUE  | TRUE  |
| delete-object                                   | DELETE | HandleDelete             | 1       | TRUE  | TRUE  |
| delete-objects                                  | POST   | HandleModifyBucket       | 1       | TRUE  | TRUE  |
| get-object                                      | GET    | HandleDownload           | 1       | TRUE  | TRUE  |
| head-bucket                                     | HEAD   | HandleBucketHead         | 1       | TRUE  | TRUE  |
| head-object                                     | HEAD   | HandleHeadObject         | 1       | TRUE  | TRUE  |
| list-buckets                                    | GET    | HandleListBuckets        | 1       | TRUE  | TRUE  |
| list-objects                                    | GET    | HandleBucket             | 1       | TRUE  | TRUE  |
| list-objects-v2                                 | GET    | HandleBucket             | 1       | TRUE  | TRUE  |
| put-object                                      | PUT    | HandleUpload             | 1       | TRUE  | TRUE  |
| upload-part                                     | PUT    | HandleUpload             | 2       | FALSE | FALSE |
| abort-multipart-upload                          | DELETE | HandleDelete             | 2       | FALSE | FALSE |
| complete-multipart-upload                       | POST   | HandleUploadModification | 2       | FALSE | FALSE |
| create-multipart-upload                         | POST   | HandleUploadModification | 2       | FALSE | FALSE |
| delete-bucket-policy                            | DELETE | HandleDeleteBucket       | 3       | FALSE | FALSE |
| delete-bucket-tagging                           | DELETE | HandleDeleteBucket       | 1       | TRUE  | TRUE  |
| delete-object-tagging                           | DELETE | HandleDelete             | 1       | TRUE  | TRUE  |
| delete-public-access-block                      | DELETE | HandleDeleteBucket       | 2       | FALSE | FALSE |
| get-bucket-acl                                  | GET    | HandleBucket             | 1       | TRUE  | TRUE  |
| get-bucket-location                             | GET    | HandleBucket             | 1       | TRUE  | TRUE  |
| get-bucket-policy                               | GET    | HandleBucket             | 3       | TRUE  | TRUE  |
| get-bucket-tagging                              | GET    | HandleBucket             | 1       | TRUE  | TRUE  |
| get-bucket-versioning                           | GET    | HandleBucket             | 2       | FALSE | FALSE |
| get-object-acl                                  | GET    | HandleDownload           | 1       | TRUE  | TRUE  |
| get-object-tagging                              | GET    | HandleDownload           | 1       | TRUE  | TRUE  |
| get-public-access-block                         | GET    | HandleBucket             | 2       | FALSE | FALSE |
| list-multipart-uploads                          | GET    | HandleBucket             | 2       | FALSE | FALSE |
| list-object-versions                            | GET    | HandleBucket             | 2       | FALSE | FALSE |
| list-parts                                      | GET    | HandleDownload           | 2       | FALSE | FALSE |
| put-bucket-acl                                  | PUT    | HandleCreateBucket       | 3       | FALSE | FALSE |
| put-bucket-policy                               | PUT    | HandleCreateBucket       | 3       | FALSE | FALSE |
| put-bucket-tagging                              | PUT    | HandleCreateBucket       | 1       | TRUE  | TRUE  |
| put-bucket-versioning                           | PUT    | HandleCreateBucket       | 2       | FALSE | FALSE |
| put-object-acl                                  | PUT    | HandleUpload             | 3       | FALSE | FALSE |
| put-object-tagging                              | PUT    | HandleUpload             | 1       | TRUE  | TRUE  |
| put-public-access-block                         | PUT    | HandleCreateBucket       | 2       | FALSE | FALSE |
| restore-object                                  | POST   | HandleUploadModification | 4       | FALSE | FALSE |
| select-object-content                           | POST   | HandleUploadModification | 4       | FALSE | FALSE |
| upload-part-copy                                | PUT    | HandleUpload             | 2       | FALSE | FALSE |
| create-bucket-metadata-table-configuration      | PUT    | HandleModifyBucket       | 3       | FALSE | FALSE |
| create-session                                  | GET    | HandleBucket             | 2       | FALSE | FALSE |
| delete-bucket-analytics-configuration           | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-cors                              | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-encryption                        | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-intelligent-tiering-configuration | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-inventory-configuration           | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-lifecycle                         | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-metadata-table-configuration      | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-metrics-configuration             | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-ownership-controls                | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-replication                       | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| delete-bucket-website                           | DELETE | HandleDeleteBucket       | 4       | FALSE | FALSE |
| get-bucket-accelerate-configuration             | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-analytics-configuration              | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-cors                                 | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-encryption                           | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-intelligent-tiering-configuration    | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-inventory-configuration              | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-lifecycle-configuration              | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-logging                              | GET    | HandleBucket             | 3       | FALSE | FALSE |
| get-bucket-metadata-table-configuration         | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-metrics-configuration                | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-notification-configuration           | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-ownership-controls                   | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-policy-status                        | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-replication                          | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-request-payment                      | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-bucket-website                              | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-object-attributes                           | GET    | HandleDownload           | 1       | TRUE  | TRUE  |
| get-object-legal-hold                           | GET    | HandleDownload           | 4       | FALSE | FALSE |
| get-object-lock-configuration                   | GET    | HandleBucket             | 4       | FALSE | FALSE |
| get-object-retention                            | GET    | HandleDownload           | 4       | FALSE | FALSE |
| get-object-torrent                              | GET    | HandleDownload           | 4       | FALSE | FALSE |
| list-bucket-analytics-configurations            | GET    | HandleBucket             | 4       | FALSE | FALSE |
| list-bucket-intelligent-tiering-configurations  | GET    | HandleBucket             | 4       | FALSE | FALSE |
| list-bucket-inventory-configurations            | GET    | HandleBucket             | 4       | FALSE | FALSE |
| list-bucket-metrics-configurations              | GET    | HandleBucket             | 4       | FALSE | FALSE |
| list-directory-buckets                          | GET    | HandleListBuckets        | 4       | TRUE  | TRUE  |
| put-bucket-accelerate-configuration             | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-analytics-configuration              | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-cors                                 | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-encryption                           | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-intelligent-tiering-configuration    | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-inventory-configuration              | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-lifecycle-configuration              | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-logging                              | PUT    | HandleCreateBucket       | 3       | FALSE | FALSE |
| put-bucket-metrics-configuration                | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-notification-configuration           | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-ownership-controls                   | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-replication                          | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-request-payment                      | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-bucket-website                              | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-object-legal-hold                           | PUT    | HandleUpload             | 4       | FALSE | FALSE |
| put-object-lock-configuration                   | PUT    | HandleCreateBucket       | 4       | FALSE | FALSE |
| put-object-retention                            | PUT    | HandleUpload             | 4       | FALSE | FALSE |
