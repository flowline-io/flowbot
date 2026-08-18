# Agent Note: Pin CI setup-go to Go 1.26.6

Status: implemented

## Problem

`actions/setup-go` configured with `go-version: '^1.26'` may resolve to any cached 1.26.x patch on the runner. When `go.mod` requires `go 1.26.6`, jobs can still start with 1.26.5 and fail at `go mod download` under `GOTOOLCHAIN=local`.

## Decision

GitHub Actions workflows that run `go mod download` or Go build/test tasks use `actions/setup-go@v7` with `go-version: '1.26.6'` instead of `'^1.26'`.

## Alternatives considered

- Keep `'^1.26'` and rely on runner cache freshness.
  - Rejected because cached patch versions are not deterministic across runners.
- Keep `'^1.26'` and remove local toolchain constraints.
  - Rejected because this weakens hermetic behavior and still does not guarantee 1.26.6 at job start.

## Consequences

- CI jobs become deterministic for patch-level Go resolution.
- `go.mod` language version and runner Go version stay aligned without depending on cache churn.
- Patch updates require explicit workflow edits when upgrading beyond 1.26.6.

## Verification

- `.github/workflows/build.yml`
- `.github/workflows/testing.yml`
- `.github/workflows/build_cli.yml`
- `.github/workflows/build_agent.yml`
- `.github/workflows/build_gateway.yml`
- `.github/workflows/agent-eval.yml`
- `.github/workflows/release.yml`
