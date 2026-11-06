# Agent Note: Lineage flows live in Analytics

Status: implemented

## Problem

M4.5 grilling planned a separate `flow_edge_daily` rollup, `retention.flow_daily_days`, a dedicated lineage module for REST, and a standalone `/service/web/flows` page. Analytics already ships live store aggregation for Personal and Ops with no rollup tables. Building a second fact store for the same window sizes (24h / 7d / 30d) would fork retention, ownership, and UI shell before `resource_links` itself is generalized (T45.2).

## Decision

`flow.graph` aggregates edges in real time from `resource_links` (and static edges from published definitions when requested). There is no `flow_edge_daily` table, no incremental daily job, and no histogram rollup store.

`/flows` is the third Analytics partition beside Personal and Ops: the User-nav shell is Personal / Ops / Flows. Flows reuses the Analytics time-range presets and module boundary; it is not a separate top-level product surface.

D17: `retention.flow_daily_days` is deleted. Edge lifetime follows `data_events` retention and the cascade that already removes `resource_links`; there is no separate flow-daily retention knob.

D15 layering: `internal/store` owns the SQL and query shapes; the analytics domain service owns `flow.graph` (and related read orchestration); MCP tools call that service; REST for flows/lineage graph reads is mounted by the analytics module (not a revived `modules/lineage`, and not hub).

T45.2 (`resource_links` generalization and backfill) and later lineage tasks proceed only under this note. They must not introduce `flow_edge_daily` or restore `retention.flow_daily_days`. The schema slice that shipped is [2026-11-07-resource-links-generalization.md](../feature/2026-11-07-resource-links-generalization.md).

## Alternatives considered

- Keep `flow_edge_daily` plus a 5-minute aggregator so graphs survive after `data_events` expire — rejected. Analytics v1 already chose live queries over rollups; the UI windows fit inside typical event retention, and a second store duplicates uid/visibility rules.
- Standalone `/service/web/flows` outside Analytics — rejected. Flows is the same operator question as Ops (what moved where), and a third User-nav shell would duplicate range controls and nav.
- Keep `retention.flow_daily_days` as a no-op until a rollup appears — rejected under pre-release foundation-over-shims; a config field for a table that must not exist is a footgun.
- Revive `internal/modules/lineage` for REST while MCP calls store directly — rejected. Audit already left that module ([merge note](../simplification/2026-10-21-merge-lineage-module-into-web.md)); MCP and REST must share the analytics service, and store stays query-only for HTTP/MCP callers.

## Consequences

- The earlier M4.5 plan items that named `flow_edge_daily`, `FlowDailyRetentionDays`, and a lineage-module mount are superseded by this note.
- [2026-10-17-mcp-rate-limit-and-audit-list.md](../feature/2026-10-17-mcp-rate-limit-and-audit-list.md) no longer treats `retention.flow_daily_days` as forthcoming product surface.
- [2026-10-21-analytics-replaces-life.md](../feature/2026-10-21-analytics-replaces-life.md) Personal+Ops shell gains Flows as the third partition when that UI lands.
- After event retention deletes old `data_events` / `resource_links`, `flow.graph` cannot show counts beyond what remains; there is no daily rollup to extend the window.
- Config, reference YAML, and i18n entries for `retention.flow_daily_days` are removed in the same change that implements Flows / deletes the dead field.

## Verification

- This file is present on `master` at `.agents/notes/implemented/architecture/2026-10-30-lineage-flows-in-analytics.md` (raw/GitHub HTTP 200).
- The Decision section states all four locks in plain sentences: live `flow.graph` (no `flow_edge_daily`); Flows as Analytics Personal/Ops/Flows; delete `retention.flow_daily_days`; store query / analytics service / MCP→service / analytics-mounted REST.
- T45.2 worktrees and checklists treat this note as the gate: no `flow_edge_daily` schema or retention field in that change.
- Owning product docs stay at [docs/analytics/README.md](../../../docs/analytics/README.md) once Flows ships; this note owns the trade-off.
