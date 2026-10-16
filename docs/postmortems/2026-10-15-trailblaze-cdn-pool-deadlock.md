# Post-mortem: cdn.trailblaze.to hung for 15 h 16 m (2026-10-15)

| | |
|---|---|
| **Status** | Resolved; all action items done |
| **Severity** | High: production asset CDN unavailable |
| **Duration** | 04:28:34 → 19:44:51 UTC, 2026-10-15 (15 h 16 m) |
| **Affected** | `trailblaze-openbucket` (Lattice stack 26): every request touching its database, including all public downloads from cdn.trailblaze.to; console listings of Trailblaze sessions |
| **Not affected** | APLB storage (cdn.appleby.cloud, stack 24), openbucket-api for APLB sessions |
| **Incident issue** | #33 |

## Summary

During a test of the console's new "Load more" feature, openbucket-web requested
metadata (bulk HEAD) for 10,000 objects in Trailblaze's `listings/banners/`.
openbucket-api fanned this out to ~400 simultaneous HEADs at the Trailblaze
storage server. A latent bug in openbucket-go made every object request hold one
pooled database connection while waiting for a second, so ~100 concurrent requests
exhausted the 100-connection pool and waited on each other forever. Nothing in the
stack timed out, so the server stayed hung — still accepting and logging requests,
completing none — until it was restarted 15 hours later. No alert existed to notice.

## Timeline (UTC)

**2026-10-15**

| Time | Event |
|---|---|
| 04:22 | openbucket-api deployed with LAN routing to storage (openbucket-api#25). API → storage calls stop passing through Cloudflare. |
| 04:28:02 | Console lists `listings/banners/` (10,000 entries) — 200 in 1.3 s. |
| 04:28:34 | Console sends bulk HEADs for those 10,000 objects (100-key batches, 4 in parallel; the API runs each batch as 100 concurrent storage calls). Last request the storage server completes. |
| 04:28–04:55 | Further console requests time out; a 4-object root listing takes 9.7 s and fails. |
| 04:55 | Detected by the agent running the test: console timeout → storage logs (requests logged, none completed) → cdn.trailblaze.to returning nothing after 20 s. |
| 04:57 | Restart of the storage app container requested from the owner (production action, approval required). |
| 19:44 | Approval received; container 173 restarted at 19:44:51. cdn.trailblaze.to back (200 in ~0.5 s). |
| ~19:50 | Root cause identified and reproduced in a test. |
| 23:34 | Fix deployed to stacks 24 and 26 (#32). |
| 23:36 | API concurrency caps deployed (openbucket-api#26). |
| 23:39 | The triggering load (10,000-object bulk HEAD, same parallelism) re-run against production with a CDN watchdog every 2 s: 100/100 batches OK, 57/57 CDN probes 200. |

**2026-10-16**

| Time | Event |
|---|---|
| 01:46 | Query deadlines and `GET /_health` live on both instances (#34). |
| 16:32 | Docker `HEALTHCHECK` live on both instances; Lattice reports `healthy` (#35). |
| 17:54 | Storage health events flowing to Monitor; alert rules enabled (openbucket-api#27–#30). |

## Root cause

`objects.GetObject` read an object's tags with a second query **inside** its
`rows.Next()` loop, while the first query's rows — and their connection — were still
open. Every object request runs `GetObject` (the permissions middleware loads object
metadata, so public downloads included). Each request therefore needed two pooled
connections at once. With `SetMaxOpenConns(100)` and ~100+ concurrent object
requests, every connection was held by a request waiting for another: a classic pool
deadlock. `bucket.ListBuckets` had the same pattern.

`database/sql` waits for a free connection indefinitely and nothing above it had a
deadline, so the deadlock never broke on its own.

## Contributing factors

1. **No timeouts on database access.** One stuck pool stopped the whole server for
   good instead of failing some requests.
2. **Unbounded fan-out in the API.** Bulk HEAD/ACL started all 100 storage calls per
   request at once, the console sent 4 such requests in parallel, and the storage
   HTTP client had no per-host connection limit.
3. **Removing an accidental throttle.** Before LAN routing, API → storage traffic
   crossed Cloudflare and the requests arrived spread out. The bug predates the
   change; the faster path made the burst land at once.
4. **No detection.** No OpenBucket service reported to Monitor, and the container
   had no health check. The server answered requests that don't touch the database
   (an unsigned `GET /` returned 403 quickly), so a liveness check would have stayed
   green anyway.
5. **Load testing against production.** The 10,000-object test ran against the
   production Trailblaze instance, with no watchdog, and testing continued after the
   first timeout instead of stopping to check the storage server.
6. **Mitigation waited on a single approver.** A restart needed explicit approval and
   none was available for ~15 h; there was no pre-approved runbook for restarting a
   hung storage server.

## What went well

- Once investigated, the cause was clear from the logs within minutes, and the bug
  was reproduced in a test (2-connection pool, 20 concurrent calls) before any fix
  shipped.
- The restart restored service immediately; no data was lost (writes simply failed).
- The fix was verified by re-running the exact triggering load against production
  under a watchdog.

## Action items

| Item | Status |
|---|---|
| Never query while rows are open: `GetObject`, `ListBuckets`; regression test `TestGetObjectUnderPoolPressure` | Done — #32 |
| Cap API → storage concurrency: 16 per bulk request, 48 connections per storage server | Done — openbucket-api#26 |
| Deadline on every query (30 s incl. pool acquire), so exhaustion fails instead of hanging | Done — #34 |
| `GET /_health` that needs a working pooled DB connection within 2 s | Done — #34 |
| Docker `HEALTHCHECK` on `/_health` (Lattice health-gates deploys, flags unhealthy) | Done — #35 |
| Storage health reported to Monitor every 30 s (public + LAN); alerts "openbucket storage unhealthy" (P1) and "openbucket storage probe silent" (P2) | Done — openbucket-api#27–#30 |
| Rule in openbucket-go AGENTS.md: never run a query while a `*sql.Rows` is open | Done — #32 |

## Open questions for the owner

- **Pre-approved remediation.** Should restarting a storage app container that fails
  its health check be allowed without a per-incident approval (or be automated —
  Lattice does not restart unhealthy containers today)? The 15-hour duration was
  almost entirely the wait for approval.
- **Load tests.** Large-volume tests should run against a non-production instance, or
  against production only with a watchdog and a stop condition agreed beforehand.

## Lessons

- A connection pool without acquire deadlines turns any pool-exhaustion bug into a
  total, permanent outage.
- Health checks must exercise the dependency that fails; "process is up" and "answers
  HTTP" were both true for the whole outage.
- Making a path faster can expose latent concurrency bugs that latency was hiding.
