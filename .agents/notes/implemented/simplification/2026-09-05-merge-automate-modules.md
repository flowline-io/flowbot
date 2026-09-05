# Agent Note: Merge functions/pipeline/workflow into automate

Status: implemented

## Problem

Named functions, pipeline, and workflow each had a near-identical interaction module (`internal/modules/{functions,pipeline,workflow}`) — same Init/enabled pattern, thin JSON handlers over an active service, and separate fx + config entries. That tripled boilerplate without a domain boundary worth three module identities.

## Decision

One module package `internal/modules/automate` (`Name = "automate"`) owns all three REST surfaces:

- Mount groups: `/service/automate/functions`, `/service/automate/pipeline`, `/service/automate/workflow`
- Scope strings stay `function:*` / `pipeline:*` / `workflow:*`; `auth.MinimumServiceScope` keys off the new groups (`automate/functions`, …)
- Config: single `modules.automate.enabled` (reference config replaces the old `workflow` / `pipeline` entries)
- `pkg/client`, call-URL helpers, capability description, and generated skills use the new prefixes
- Web HTML under `/service/web/*` is unchanged

Deleted packages: `internal/modules/functions`, `pipeline`, `workflow`.

## Alternatives considered

- Merge into `internal/modules/web` — rejected; web is HTML/HTMX ops console, not the CLI/REST module boundary
- Keep historical `/service/{functions|pipeline|workflow}` prefixes — rejected for this change; pre-release prefers one automate URL tree over shims
- Unify scopes to `automate:*` — rejected; existing token scopes remain useful per domain

## Consequences

- Existing clients and tokens that call old `/service/functions|pipeline|workflow` paths must update URLs (scopes unchanged)
- Orphan `modules.workflow` / `modules.pipeline` config keys are ignored after upgrade

## Verification

- `internal/modules/automate` mounts `automate/functions|pipeline|workflow`; `fx.go` registers only `automate.Register`
- `pkg/auth.MinimumServiceScope` maps those groups to `function:*` / `pipeline:*` / `workflow:*`; clients and call-URL helpers use `/service/automate/...`
- Capability description in `pkg/capability/functions` feeds `docs/skills` via `go tool task skills`
- Package tests: `go test ./internal/modules/automate/... ./pkg/auth/... ./pkg/client/... ./pkg/views/partials/`
