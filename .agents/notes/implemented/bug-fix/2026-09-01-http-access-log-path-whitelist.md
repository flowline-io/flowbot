# Agent Note: HTTP access log path whitelist

Status: implemented

## Problem

Internet scanners hammer the origin with `GET /.env`, `/admin/.env`, and similar paths. Fiber zerolog logged every 4xx as warn `Client error`, so production logs were mostly unmatched-route 404s. Real 404s on the HTTP surface (broken clients, missing hub apps, bad static URLs) were drowned out.

## Decision

Access logs emit only for a hardcoded list of top-level path prefixes that match the HTTP surface: `/service`, `/hub`, `/chatagent`, `/gateway`, `/static`, `/platform`, `/oauth`, `/webhook`, `/agent`, `/form`, `/c`, `/swagger`. A path matches a prefix when it equals the prefix or starts with `prefix+"/"` (`/agent` logs, `/agentfoo` does not).

Off-whitelist paths skip the access log entirely (not debug, not sampled). Quiet exact paths still skip even when they sit under a prefix: `/`, health checks, `/metrics`, `/service/user/metrics`. Domain `types.ErrNotFound` still goes through the ErrorHandler `flog.Error` path; this change is access-log only.

`shouldSkipAccessLog` / `skipAccessLogPath` in `internal/server/http.go` own the list.

## Alternatives considered

- **Skip every HTTP 404** — hides broken clients on real routes.
- **Skip only unmatched Fiber routes** — logger `Next` runs before route resolution.
- **Configurable prefix list in `flowbot.yaml`** — duplicates the route tree already in `router.go`.
- **Downgrade off-whitelist 404s to debug** — still fills the debug sink when that level is on.
- **Probe-path denylist (`.env`, `wp-admin`)** — incomplete and unbounded.

## Consequences

Probes such as `/.env.yml` and `/admin/.env` no longer appear. `/hub/.env` and `/static/.env` still log because the prefix matches. `/static/*` 200s stay at info, same as before the whitelist.

## Verification

- `go test ./internal/server/ -run 'TestSkipAccessLogPath|TestShouldSkipRateLimit'`
- `TestSkipAccessLogPath` covers scanner roots, segment boundaries, quiet overlay paths, and each whitelist prefix.
