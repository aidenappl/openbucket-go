CREATE TYPE "acl_type" AS ENUM (
  'PRIVATE',
  'PUBLIC_READ',
  'PUBLIC_READ_WRITE',
  'AUTHENTICATED_READ',
  'BUCKET_OWNER_READ',
  'BUCKET_OWNER_FULL_CONTROL',
  'LOG_DELIVERY_WRITE'
);

CREATE TYPE "permission_type" AS ENUM (
  'FULL_CONTROL',
  'READ',
  'WRITE',
  'READ_ACP',
  'WRITE_ACP'
);

CREATE TABLE "authorizations" (
  "id" serial PRIMARY KEY,
  "name" text NOT NULL,
  "key_id" varchar(64) UNIQUE NOT NULL,
  "secret_key" varchar(128) NOT NULL,
  "date_created" timestamp DEFAULT (now())
);

CREATE TABLE "buckets" (
  "id" serial PRIMARY KEY,
  "name" text UNIQUE NOT NULL,
  "creation_date" timestamp DEFAULT (now()),
  "acl" acl_type DEFAULT 'PRIVATE',
  "owner_id" int
);

CREATE TABLE "bucket_permissions" (
  "id" serial PRIMARY KEY,
  "bucket_id" int,
  "grantee_id" int,
  "permission" permission_type NOT NULL,
  "date_added" timestamp DEFAULT (now())
);

CREATE TABLE "object_tags" (
  "id" serial PRIMARY KEY,
  "bucket_id" int,
  "object_id" int,
  "tag_key" text NOT NULL,
  "tag_value" text NOT NULL
);

CREATE TABLE "bucket_tags" (
  "id" serial PRIMARY KEY,
  "bucket_id" int,
  "tag_key" text NOT NULL,
  "tag_value" text NOT NULL
);

CREATE TABLE "objects" (
  "id" serial PRIMARY KEY,
  "bucket_id" int,
  "key" text NOT NULL,
  "etag" varchar(64) NOT NULL,
  "version_id" int DEFAULT 1,
  "owner_id" int,
  "public" boolean DEFAULT false,
  "size" bigint DEFAULT 0,
  "last_modified" timestamp NOT NULL,
  "uploaded_at" timestamp NOT NULL
);

ALTER TABLE "buckets" ADD FOREIGN KEY ("owner_id") REFERENCES "authorizations" ("id");

ALTER TABLE "bucket_permissions" ADD FOREIGN KEY ("bucket_id") REFERENCES "buckets" ("id");

ALTER TABLE "bucket_permissions" ADD FOREIGN KEY ("grantee_id") REFERENCES "authorizations" ("id");

ALTER TABLE "object_tags" ADD FOREIGN KEY ("bucket_id") REFERENCES "buckets" ("id");

ALTER TABLE "object_tags" ADD FOREIGN KEY ("object_id") REFERENCES "objects" ("id");

ALTER TABLE "bucket_tags" ADD FOREIGN KEY ("bucket_id") REFERENCES "buckets" ("id");

ALTER TABLE "objects" ADD FOREIGN KEY ("bucket_id") REFERENCES "buckets" ("id");

ALTER TABLE "objects" ADD FOREIGN KEY ("owner_id") REFERENCES "authorizations" ("id");
