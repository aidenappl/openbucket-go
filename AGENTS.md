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
tools/, util/        ETags, content types, ACL strings, path validation
cli/                 CLI commands (setup, backup, …)
types/               S3 XML structs and metadata types
init.sql             reference DDL (RunMigrations is the source of truth)
```

## Running, building & testing

```bash
dev up                 # docker compose: Postgres + server (localhost:8080)
dev build | dev test | dev vet | dev check
go test ./listing ./responder ./types                 # unit tests, no env needed
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
`--endpoint-url http://localhost:8080` and region `openbucket`.

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
- **Delete** is idempotent; deleting a folder marker removes only the marker.

## Ecosystem & related repos

- **openbucket-api** is the main client (aws-sdk-go v1): every console listing, upload, delete,
  HEAD and presign goes through this server. Its folder delete/rename loop over `ListObjectsV2`.
- **openbucket-web** → openbucket-api → this server.
- **openbucket-cli**, **openbucket-mcp** reach it through openbucket-api.
- Atlas and Trailblaze serve public assets from it via the CDNs below.

## Operations

- One image, `registry.appleby.cloud/openbucket:latest`, built by
  `.github/workflows/push-to-registry.yml` on push to `main`. **CI does not deploy.**
- Two production instances, each its own Lattice stack (server + `postgres:16` with the `db_data`
  volume, files at `/opt/openbucket/buckets`):
  - **stack 24** "APLB OpenBucket Core" — `aplb-openbucket`, `cdn.appleby.cloud` (Atlas, APLB)
  - **stack 26** "Trailblaze OpenBucket Core" — `trailblaze-openbucket`, `cdn.trailblaze.to`
- Deploy = `lattice_deploy_stack` (rolling) per stack, APLB first, then Trailblaze, verifying
  listings and a public CDN object after each. Logs: `lattice_get_container_logs`.
- Migrations run on every boot (`db.RunMigrations`, `IF NOT EXISTS`).

## Rules & guardrails

- Don't change listing order or prefix matching without running the integration test: byte order
  is what keeps pages from losing or repeating keys.
- Never call `objects.ListObjects` with `Keys: nil` meaning "these keys" — nil is "no filter".
- Don't reintroduce `os.RemoveAll` for folder markers.
- Deploying touches both production storage instances — ask first, and roll one stack at a time.

## Verification

```bash
gofmt -l .            # must print nothing for files you touched
go build ./... && go vet ./...
go test ./listing ./responder ./types
go test -tags integration ./objects   # when touching objects/, listing/ or db/
```

## Keeping this file updated

Any change to structure, stack, commands, conventions, routes, schema or S3 behaviour updates this
file (and `README.md` where setup or behaviour changed) in the same change.
