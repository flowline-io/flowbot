# Agent Note: CLI version prints client and server

Status: implemented

## Problem

`flowbot version` only printed the local CLI build tag. Operators comparing a installed CLI against a running Flowbot instance had no single command to see both sides.

## Decision

- Expose public `GET /version` on the HTTP server (next to `/livez` / `/readyz`), returning `version.Buildtags` and `version.Buildstamp` in a protocol success payload. No auth; quiet for rate-limit and access-log skip lists.
- CLI `version` always prints `Client version: …`, then best-effort fetches the server via `utils.NewPublicClient` (server URL only, no token). On missing URL or request failure it prints `Server version: unavailable (…)` and still exits 0.

## Alternatives considered

- Reuse `/chatagent/info` — rejected; requires chatagent scope and returns agent splash metadata unrelated to build identity.
- Call removed `/service/server/stacktrace` for `build_info.main_version` — rejected; server module is gone, and stacktrace is a heavy diagnostics surface.
- Fail `version` when the server is unreachable — rejected; install scripts and offline CLI use rely on a successful local version print.

## Consequences

- `flowbot version` output is two labeled lines instead of `flowbot version <tag>`.
- Public `/version` discloses build tags to anyone who can reach the HTTP port (same sensitivity as an About page).

## Verification

- `go test ./internal/server -run 'TestVersionHandler|TestRouteRegistration|TestShouldSkip'`
- `go test ./pkg/client -run TestClientVersion`
- `go test ./cmd/cli/command -run TestVersion`
