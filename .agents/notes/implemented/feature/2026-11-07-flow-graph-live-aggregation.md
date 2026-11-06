# Agent Note: flow.graph live aggregation and multi-hop Trace

Status: implemented

## Problem

M4.5 needed multi-hop lineage reads and a flow topology API. The earlier plan introduced `flow_edge_daily` and `retention.flow_daily_days`, which [lineage-flows-in-analytics](../architecture/2026-10-30-lineage-flows-in-analytics.md) rejected in favor of live aggregation over `resource_links`. Without store queries and an analytics-owned service, T45.5 MCP tools and the Flows UI have nothing to call.

## Decision

`ResourceChainStore.AggregateFlowGraph` groups `resource_links` in a time window by `(source_app, target_app, run_kind, pipeline_name)` with a filtered Select and in-memory fold (no daily table; avoids SQLite/`Max(time)` scan mismatches). Window defaults to 7 days, capped by `min(retention.data_events_days if > 0, 90)`.

`ResourceChainStore.Trace` walks edges from `event_id`, `app+entity_id`, or `correlation_id` with depth ≤ 10 and ≤ 500 nodes (`truncated` when either limit bites). Default direction is `up`. Expansion is a bounded BFS over Ent neighbor queries (behavior-equivalent to a recursive CTE without exposing `*sql.DB` through the Adapter).

`analytics.Service` owns `FlowGraph` and `Trace` orchestration (D15). Static edges and MCP/REST/CLI stay out of this slice; `include_static` is accepted and ignored.

D17: `retention.flow_daily_days` and `FlowDailyRetentionDays` are removed from config, reference YAML, and generated field docs. Leftover YAML keys are ignored by mapstructure.

## Alternatives considered

- Ship `flow_edge_daily` for post-retention graph survival — rejected; locked by the lineage-flows architecture note.
- Literal `WITH RECURSIVE` via a new Adapter `SQLDB()` — deferred; Ent Client does not expose the dialect driver to domain stores, and BFS meets the same depth/node contract for sqlite tests and production.
- Ent `GroupBy` + `Max(created_at)` — rejected for this slice; SQLite returns the max timestamp as a string and breaks `time.Time` Scan.
- Include static edges from published pipeline definitions — deferred (cuttable A″); no capability→app registry is ready for reliable source apps.

## Consequences

- MCP `flow.graph` / `lineage.trace` (T45.5) call analytics service methods, not store directly.
- After `data_events` retention cascades `resource_links`, graph counts cannot exceed what remains.
- Operators with a leftover `retention.flow_daily_days` key see no behavior change; the key is ignored.

## Verification

- `go test ./internal/store/ -run 'TestAggregateFlowGraph|TestTrace_'` and `go test ./internal/modules/analytics/ -run FlowGraph|Trace` and `go test ./pkg/types/ -run NormalizeFlowGraph` pass.
- Grep finds no `flow_edge_daily` schema or `FlowDailyDays` field.
- `CHANGELOG.md` Unreleased lists Added (store/service) and Removed (`retention.flow_daily_days`).
