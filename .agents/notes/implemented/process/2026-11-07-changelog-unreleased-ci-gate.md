# Agent Note: Changelog Unreleased CI gate

Status: implemented

## Problem

Removals of providers, capabilities, modules, or commands, and contract edits to auth scopes, reference-config top-level keys, or the MCP tool snapshot, can land without a `CHANGELOG.md` `## Unreleased` bullet. Operators and release notes then miss the break. PR checklist text alone does not catch that.

## Decision

CI runs `scripts/check-changelog-unreleased.sh` (`go tool task check:changelog`) as the `changelog` job on `.github/workflows/mcp-contract.yml`.

The script diffs `HEAD` against `BASE_SHA` (PR base SHA, else push `before`, else `merge-base` with `origin/master` / `master`). It treats these as contract-sensitive triggers:

- An immediate child directory under `pkg/providers/`, `pkg/capability/`, `internal/modules/`, or `cmd/` present at the base and absent at `HEAD`
- A change to string or alias assignments inside the `const (` block of `pkg/auth/scope.go`
- A change to uncommented top-level keys in `docs/reference/config.yaml`
- Any change to `internal/server/mcp/testdata/tools.snapshot.json`

When any trigger fires and `## Unreleased` has no new non-empty line relative to the base, the job emits a GitHub `::warning::` and exits 0 through 2026-11-15 UTC. From **2026-11-16** UTC (`BLOCK_ON`, overridable) the same condition is a hard failure (`::error::`, exit 1). That date is the M5 cutover from warn to block.

## Alternatives considered

- **Dedicated workflow only.** Rejected as the default: mcp-contract already runs on every push/PR without `paths-ignore`, and the snapshot trigger sits next to the MCP contract tests. A separate workflow remains possible later if the job outgrows this file.
- **Require an Unreleased line on every PR.** Rejected: most PRs are not contract-sensitive; the trigger list keeps noise low.
- **Block immediately.** Rejected until M5 so existing in-flight removals can adopt the habit under warning first.

## Consequences

- Authors who delete a watched package directory or change scopes / reference top-level keys / the tool snapshot must add a non-empty Unreleased line in the same change, or accept a warning (then a red check after 2026-11-16).
- Nested file edits inside a watched tree without removing the directory do not trip the directory trigger.
- Comment-only or nested-key-only edits to `scope.go` / `config.yaml` do not trip those triggers.

## Verification

- `.github/workflows/mcp-contract.yml` defines a `changelog` job with `fetch-depth: 0` that runs `bash scripts/check-changelog-unreleased.sh`.
- `go tool task check:changelog` invokes the same script.
- With `BASE_SHA` set to a parent that has a trigger and an unchanged Unreleased section, the script prints a warning and exits 0 while the UTC date is before 2026-11-16; with `BLOCK_ON` set to a past date it exits 1.
