# Agent Note: Analytics replaces Life

Status: implemented

## Problem

Life was a full solo RPG productivity vertical (goals, quests, loot, CapLife) that competed with Flowbot's homelab orchestration focus. Usage and run metrics already existed as scattered surfaces (home token usage, pipeline/workflow/function stats panels) without a coherent User-nav product.

## Decision

Hard-delete Life (code, `life_*` schemas, CapLife, web UI, docs, BDD) and ship `internal/modules/analytics` as the User-nav successor: one shell with Personal (default) and Ops sections. v1 uses live store aggregation only (no capability, no `analytics_*` rollup tables). Token usage lives at `/service/web/analytics/token-usage`; `/home/token-usage` redirects. Personal scopes by uid (events, tokens, owned pipeline/function runs); Ops is site-wide. Workflow definitions lack `created_by`, so Personal Activity covers pipeline and function runs only.

## Alternatives considered

- Soft-freeze Life / leave schemas — rejected under pre-release foundation-over-shims; half-dead Life pollutes nav, store, and capability registration.
- Web-only pages without `modules/analytics` — rejected; Personal+Ops needs a module boundary symmetric with Life's removal and room for later CapAnalytics.
- CapAnalytics in v1 — rejected; stabilize UI query contracts before locking operation names.
- Rollup tables in v1 — rejected; validate which charts matter before inventing a second fact store.

## Consequences

- Deployed databases keep orphan `life_*` tables until operators `DROP` them; Ent `Schema.Create` does not remove them.
- Existing bookmarks to `/service/web/life*` and Life notify template IDs (`life.quest.*`) stop resolving; home token embed is a link into analytics.
- Pipeline/workflow/function list pages keep their own stats panels; analytics Ops is the cross-kind health surface under User nav.

## Verification

Package tests cover analytics web handlers and store aggregations (`./internal/modules/web/`, `./internal/store/`, `./pkg/types/`). Lint (`go tool task lint`) and `go tool task build` succeed with CapLife and `modules/life` absent from the graph. Owning docs: `docs/analytics/README.md`.
