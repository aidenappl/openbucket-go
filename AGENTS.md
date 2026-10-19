# AGENTS.md — openbucket-go

## What this repo is

The OpenBucket storage server: an S3-compatible object store written in Go. It speaks the S3 REST
API (SigV4 auth, XML) for buckets and objects, stores object **bytes as files on disk**
(`buckets/<bucket>/<key>`) and **metadata in Postgres** (keys, ETags, owners, ACLs, grants, tags),
and exposes a small JSON admin API for credentials, buckets and grants. The same binary is also a
CLI (`./openbucket <command>`).

It owns storage and S3 semantics. It does **not** own users, sessions or the web console — those
are openbucket-api (which talks to this server as an S3 client) and openbucket-web.

## Stack & dependencies

- Go 1.26 (`go.mod`), `gorilla/mux`, `Masterminds/squirrel` (Postgres placeholders, `db.Psql`),
  `lib/pq`.
- **Postgres 16**, schema `core` (`search_path=core` in `DATABASE`). Not MariaDB — this repo is a
  deliberate exception to the global default.
- No internal SDKs; SigV4 verification is in-repo (`aws/ValidateSignature.aws.go`).

## Project structure

```
main.go              server (no args) or CLI (any args); routes
env/                 PORT, REGION, DATABASE, PRESIGN_SECRET_KEY, PRESIGN_BASE_URL, OB_ADMIN_TOKEN
db/                  connection pool, RunMigrations (idempotent schema + indexes, every boot)
aws/                 SigV4 signature validation
auth/                credentials (authorizations table), permission checks + cache
bucket/              bucket CRUD, grants, tags, stats; bucket lookup cache (2 min)
objects/             object metadata queries, ListObjectsXML (S3 listing), WalkObjects (index walk)
listing/             pure S3 paging: Paginate, tokens, Successor/GroupOf — no DB, unit-tested
metadata/            object metadata creation on upload
handler/             credential generation, grants, CopyObject
routers/             HTTP handlers (S3 + admin)
middleware/          RequestState (request/host ids), logging, AdminAuth, Authorized/HalfAuthorized
responder/           XML (S3) and JSON (admin) responses
tools/, util/        ETags (content MD5), content types, ACL strings, path validation
cli/                 CLI commands (setup, backup, backfill-etags, …)
types/               S3 XML structs and metadata types
init.sql             reference DDL (RunMigrations is the source of truth)
```

## Running, building & testing

```bash
dev up                 # docker compose: Postgres + server (localhost:8080)
dev build | dev test | dev vet | dev check
go test ./listing ./responder ./types ./tools         # unit tests, no env needed
go test -tags integration ./objects                   # needs a Postgres, see below
go test -tags integration ./objects -run '^$' -bench DelimitedPage
```

Integration tests need `DATABASE` (with `search_path=core`), `PRESIGN_SECRET_KEY` and
`OB_ADMIN_TOKEN` set — the `env` package panics without them. Use a throwaway Postgres with a
locale collation, as production has:

```bash
docker run -d --name obgo-test-db -e POSTGRES_USER=openbucket -e POSTGRES_PASSWORD=openbucket \
  -e POSTGRES_DB=openbucket -p 55433:5432 postgres:16
DATABASE='postgres://openbucket:openbucket@localhost:55433/openbucket?sslmode=disable&search_path=core' \
  PRESIGN_SECRET_KEY=test OB_ADMIN_TOKEN=test go test -tags integration ./objects
```

`go test ./...` without env fails in `middleware` (env panic) — known, not a regression.

End-to-end S3 checks: run the binary against that DB, create a credential and bucket, grant
`FULL_CONTROL` (owners need an explicit grant), and use the AWS CLI with
`--endpoint-url http://localhost:8080` and region `openbucket` — or rclone (`type=s3`,
`provider=Other`, `force_path_style=true`, region `openbucket`): `rclone copy` then
`rclone check` proves the ETags verify; `curl --aws-sigv4 "aws:amz:openbucket:s3"` works for
one-off requests.

## How code is written here

- Handlers in `routers/` are thin; S3 semantics live in `objects/`, `bucket/`, `listing/`.
- Errors to S3 clients are XML (`responder.SendXMLError`) with S3 error codes; admin errors JSON.
- **Keys are compared and ordered in byte order** (`COLLATE "C"`), never the database locale.
- Prefix filters are byte ranges (`key >= p AND key < Successor(p)`), never `LIKE` (`_` and `%`
  are wildcards).
- Pure logic goes in a package with no `db`/`env` import so it can be unit-tested (`listing/`).

## Domain & architecture

- **Auth.** S3 requests are SigV4-signed with credentials from `authorizations`. `Authorized`
  checks the signature and the bucket grant; `HalfAuthorized` only the signature (list buckets,
  create bucket). Owners need an explicit grant. Admin API: `Authorization: Bearer OB_ADMIN_TOKEN`.
- **Objects.** Upload writes the file, then the row. Folder markers are zero-byte keys ending
  in `/`.
- **Listing** (`objects/ListObjectsXML.objects.go`): S3 paging (v1 marker, v2 continuation-token
  / start-after), at most 1000 entries a page. With a delimiter, `WalkObjects` walks the
  `(bucket_id, key COLLATE "C")` index one probe per entry; `listing.Paginate` cuts the page in
  both modes.
- **CopyObject** (`handler/HandleCopyObject.handle.go`) follows S3: an existing destination is
  overwritten (it used to be refused, #23); a folder marker (`a/`) copies to a folder marker and
  only to a key ending in `/` (reading the marker's directory failed, #22 — which broke every
  folder rename); copying a key onto itself is refused unless `x-amz-metadata-directive: REPLACE`,
  and never reaches the file copy (that would truncate the file while reading it).
- **ETags are the hex MD5 of the object's bytes**, as S3 does for a single-part PUT
  (`tools/GenerateETag.tool.go`). The database stores them unquoted; every response quotes them —
  XML via `ETag.MarshalXML`, headers (PUT, HEAD, GET) via `ETag.Quoted()`. A folder marker's ETag
  is the MD5 of zero bytes. Two keys with identical bytes share an ETag, so **never treat an ETag
  as unique** (key + ETag is fine). They used to be `md5(filePath + content)`: rclone copying
  from AWS into cdn.hillview.tv flagged every object as corrupted (2026-10-18).
- **aws-chunked bodies are decoded** (`aws/Chunked.aws.go`). AWS SDK v2 clients over HTTPS send
  PutObject as `Content-Encoding: aws-chunked` / `X-Amz-Content-Sha256: STREAMING-…` with a
  trailing `x-amz-checksum-*` by default. `HandleUpload` wraps the body in `aws.NewChunkedReader`,
  which strips the framing, verifies the trailing crc32/crc32c/sha256 (`400 BadDigest`) and
  `X-Amz-Decoded-Content-Length` (`400 IncompleteBody`). Chunk signatures are parsed, not verified.
  Before this, the framing was stored inside the object (Hillview uploads, 2026-10-18). The SDK
  only does this over TLS: an end-to-end check needs HTTPS in front of the server, or the bug
  doesn't reproduce.
- **Content-MD5** on PUT is verified: a malformed header is `400 InvalidDigest`, a mismatch
  `400 BadDigest`. `CreateObject` writes to a temp file (`.upload-*`, same directory) and renames
  it into place, so a refused or failed upload never replaces the stored object.
- **Multipart upload is not implemented** (`501 NotImplemented`); clients must use single-part
  PUTs (rclone: a large `--s3-upload-cutoff`). When it is added, the completed object's ETag must
  be S3's form, `"<md5 of the concatenated binary part MD5s>-<part count>"`.
- **Delete** is idempotent; deleting a folder marker removes only the marker.
- **One row per key.** `CreateObject` / `CreateFolder` upsert (`objects/upsert.objects.go`) on the
  unique index `uniq_objects_bucket_key`. Overwrites used to insert duplicate rows; the one-time
  cleanup in `db.ensureUniqueObjectKeys` moved them to `objects_superseded` /
  `object_tags_superseded` (kept for restore).

## Ecosystem & related repos

- **openbucket-api** is the main client (aws-sdk-go v1): every console listing, upload, delete,
  HEAD and presign goes through this server. Its folder delete/rename loop over `ListObjectsV2`.
- **openbucket-web** → openbucket-api → this server.
- **openbucket-cli**, **openbucket-mcp** reach it through openbucket-api.
- Atlas and Trailblaze serve public assets from it via the CDNs below.

## Operations

- One image, `registry.appleby.cloud/openbucket:latest`, built by
  `.github/workflows/push-to-registry.yml` on push to `main`. Its `deploy` job then fires the Lattice
  automation **"Redeploy OpenBucket"** (id 2) through the `LATTICE_DEPLOY_URL` secret, which
  redeploys `hillview-openbucket`, `aplb-openbucket`, `trailblaze-openbucket` in that order. The
  automation is the instance list: a new instance is a new step there, not a workflow change.
  The job requires `data.result == "succeeded"` (a disabled automation answers 200 with
  `disabled`). A green run means Lattice accepted the redeploys, not that they landed: no step
  waits for its container, so check `lattice_list_automation_runs` and each instance.
- Three production instances, each its own Lattice stack (server + `postgres:16` with the
  `db_data` volume, files at `/opt/openbucket/buckets`):
  - **stack 24** "APLB OpenBucket Core" — `aplb-openbucket`, `cdn.appleby.cloud` (Atlas, APLB)
  - **stack 26** "Trailblaze OpenBucket Core" — `trailblaze-openbucket`, `cdn.trailblaze.to`
  - **stack 42** "Hillview OpenBucket" — `hillview-openbucket`, `cdn.hillview.tv`
- Manual deploy (CI down, or one instance only) = `lattice_deploy_stack` (rolling) per stack, or
  `lattice_run_automation` id 2 for all three, verifying listings and a public CDN object after each. Logs: `lattice_get_container_logs`.
- **ETag backfill.** Objects uploaded before the content-MD5 fix keep their old
  `md5(path + content)` ETags, so integrity-checking clients still reject them until they are
  recomputed. After deploying the fix, run once per instance, inside the server container (its
  working directory, `/app`, holds `buckets/`):
  ```bash
  ./openbucket backfill-etags --dry-run         # counts: scanned / would update / unchanged / missing
  ./openbucket backfill-etags [--bucket name]   # rewrites only the etag column; idempotent
  ```
  It reads every file once (I/O-heavy on a large instance), skips and lists rows with no file on
  disk, and leaves `last_modified` alone. Not yet run on any instance as of 2026-10-18.
- Migrations run on every boot (`db.RunMigrations`, `IF NOT EXISTS`).

## Rules & guardrails

- Don't change listing order or prefix matching without running the integration test: byte order
  is what keeps pages from losing or repeating keys.
- Never call `objects.ListObjects` with `Keys: nil` meaning "these keys" — nil is "no filter".
- Don't reintroduce `os.RemoveAll` for folder markers.
- **Every query has a deadline.** `db.Psql` runs through `db.Timed` (`QueryTimeout`, 30 s: pool
  acquire + statement + reading rows); raw SQL uses `db.Timed` too, never bare `db.DB` (migrations
  excepted). `database/sql` otherwise waits for a pooled connection forever — how the 2026-10-15
  outage hung the whole server (#33).
- **Deadlines are per request**, not server-wide: `middleware.Deadlines` (30 s read / 60 s write)
  wraps the router, and object downloads/uploads (`GET`/`PUT /{bucket}/{key}`) get
  `middleware.LongTransfer` (2 h). The server keeps only `ReadHeaderTimeout`. Server-wide
  timeouts cut every CDN download off at 60 s (a 60 MB file at 700 KB/s stopped at 45 MB).
- **Health:** `GET /_health` (unauthenticated, not logged) is 200 only when a pooled DB connection
  works within 2 s — `db.Healthy`, which enforces the 2 s itself because lib/pq's cancellation
  blocks on a frozen database. `openbucket healthcheck` exits 0/1 on it, for a container
  `HEALTHCHECK` (the image has no curl/wget) — the Dockerfile's HEALTHCHECK (every 15 s, 3
  failures) uses it, so Lattice health-gates deploys on it and flags a hung server (it does not
  restart it). openbucket-api probes it and reports to Monitor.
- **Never run a query while a `*sql.Rows` is open.** Read all rows, close them, then query
  again. `GetObject` fetched tags inside its rows loop, so every object request needed two pooled
  connections at once; ~100 concurrent requests deadlocked the pool permanently and
  cdn.trailblaze.to hung for ~15 hours (2026-10-15). `TestGetObjectUnderPoolPressure` (2-connection
  pool, 20 concurrent calls) guards it.
- Never write to `objects` with a plain INSERT — use `objectUpsert`; a second row for a key now
  fails on the unique index.
- Merging to `main` redeploys all three production storage instances within seconds of each
  other. Treat a merge as a deploy: ask first. Manual deploys roll one stack at a time.

## Verification

```bash
gofmt -l .            # must print nothing for files you touched
go build ./... && go vet ./...
go test ./listing ./responder ./types ./tools
go test -tags integration ./objects   # when touching objects/, listing/ or db/
```

Incident write-ups live in `docs/postmortems/` (2026-10-15: pool deadlock, cdn.trailblaze.to down 15 h).

## Keeping this file updated

Any change to structure, stack, commands, conventions, routes, schema or S3 behaviour updates this
file (and `README.md` where setup or behaviour changed) in the same change.
