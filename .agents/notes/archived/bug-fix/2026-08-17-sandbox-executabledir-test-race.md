# Agent Note: Sandbox executableDir test race

Status: implemented
Archived: 2026-08-22

## Problem

`go test -race ./pkg/agent/sandbox/...` failed because `TestResolveCLIBinary` subtests overwrote package-level `executableDir` under `t.Parallel()` while other parallel tests called `ResolvedCLIBinary` via `New`.

## Decision

Keep `executableDir` as a package-level test seam. Tests that override it must not use `t.Parallel()` (same pattern as other package-var overrides in this repo). Absolute `cli_path` resolution no longer calls `executableDir`.

## Alternatives considered

- **Mutex around `executableDir`** — rejected; production would pay for a test-only seam.
- **Inject dir via `Config`** — rejected for this fix; larger API change than the race requires.

## Consequences

`TestResolveCLIBinary` runs serially. Absolute configured paths resolve without depending on the server executable directory.

## Verification

`go test -race ./pkg/agent/sandbox/ -count=5` passes.
