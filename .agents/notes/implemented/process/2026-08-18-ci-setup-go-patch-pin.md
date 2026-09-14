# Agent Note: Pin CI setup-go to an exact Go patch

Status: implemented

## Problem

`actions/setup-go` configured with `go-version: '^1.27'` may resolve to any cached 1.27.x patch on the runner. When `go.mod` requires `go 1.27.1`, jobs can still start with an older 1.27.x and fail at `go mod download` under `GOTOOLCHAIN=local`.

## Decision

GitHub Actions workflows that run `go mod download` or Go build/test tasks use `actions/setup-go@v7` with `go-version: '1.27.1'` instead of `'^1.27'`. Current pin tracks [Go 1.27.1 upgrade](./2026-09-14-go-1.27.1-upgrade.md).

## Alternatives considered

- Keep `'^1.27'` and rely on runner cache freshness.
  - Rejected because cached patch versions are not deterministic across runners.
- Keep `'^1.27'` and remove local toolchain constraints.
  - Rejected because this weakens hermetic behavior and still does not guarantee 1.27.1 at job start.

## Consequences

- CI jobs become deterministic for patch-level Go resolution.
- `go.mod` language version and runner Go version stay aligned without depending on cache churn.
- Patch updates require explicit workflow edits when upgrading beyond 1.27.1.

## Verification

- `.github/workflows/build.yml`
- `.github/workflows/testing.yml`
- `.github/workflows/build_cli.yml`
- `.github/workflows/build_agent.yml`
- `.github/workflows/build_gateway.yml`
- `.github/workflows/agent-eval.yml`
- `.github/workflows/release.yml`
