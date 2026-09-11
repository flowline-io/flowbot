# Agent Note: Tighten revive rules and exclude ent/gen

Status: implemented

## Problem

`revive.toml` was already strict but lagged revive v1.15 (package checks moved out of `var-naming`) and omitted several low-noise correctness rules. `lint:go` also walked generated Ent under `internal/store/ent/gen`, unlike gosec.

## Decision

- Enable additional revive rules: `package-naming` (with grandfathering for existing `types` / `utils` / `plugin` names), `bare-return`, `early-return`, `time-equal`, `time-date`, `use-errors-new`, `unnecessary-if`, `unnecessary-format`, identical if/switch helpers, `inefficient-map-lookup`, `call-to-gc`, `use-fmt-print`, `useless-break` / `useless-fallthrough`, `redundant-build-tag` / `redundant-test-main-exit`, `confusing-results`, and `max-control-nesting` (5).
- Configure `unhandled-error` ignores for `fmt.Print*` and `bytes.Buffer.Write*`.
- Exclude `./internal/store/ent/gen/...` from `lint:go`, aligned with [.agents/notes/implemented/process/2026-09-11-gosec-g115-and-ent-gen-exclude.md](2026-09-11-gosec-g115-and-ent-gen-exclude.md).
- Keep `exported` and `var-naming` disabled (existing stance). `package-naming` uses `skip-default-bad-name-check` + `skip-collision-with-common-std` and a custom bad-name list so `pkg/types`, `pkg/utils`, and `pkg/plugin` stay legal while still banning `common` / `util` / `misc` / `interfaces` / `helpers` / `shared` / `utilities`.

## Alternatives considered

- **Enable default `package-naming` without grandfathering.** Rejected: would force renaming `pkg/types`, `pkg/utils`, and `pkg/plugin` in the same change.
- **Leave `use-errors-new` / `unnecessary-format` off.** Rejected: they match the repo error-handling style; bulk `fmt.Errorf` without verbs was mechanical to fix.
- **Re-enable `exported`.** Deferred: large godoc gap vs AGENTS requirement; separate cleanup.

## Consequences

- CI `go tool task lint` fails on naked returns, unnecessary format calls, identical switch/if branches, and confusing same-type unnamed results.
- Generated Ent is not revive-scanned; regenerate via `go tool task ent` remains the quality gate for that tree.

## Verification

```bash
go tool task lint:go
go test -count=0 ./...
```
