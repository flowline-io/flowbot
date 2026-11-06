# Lineage and flows

Operators can follow how resources move across apps: multi-hop lineage (`lineage.trace` / `lineage.search`), a live topology (`flow.graph`), and a `data_events` timeline (`events.timeline`). The same reads are available over MCP (scope `lineage:read`), REST under `/service/lineage|flows|events` (mounted by analytics), CLI (`flowbot lineage|flow|events`), and the Analytics Flows page at `/service/web/analytics/flows`.

Edges live in PostgreSQL `resource_links` (pipeline / workflow / MCP writers). There is no daily rollup table; counts come from detail edges still retained in that table. MCP tool arguments and payload rules: [mcp.md](mcp.md). Analytics UI and window presets: [Analytics](../analytics/README.md).

## Correlation

A `correlation_id` (`cor_<ULID>`) groups one run’s events, resource-link edges, and mutation audit rows. Pipeline runs stamp a missing correlation onto the trigger event and pass it into steps. Successful MCP mutation `tools/call` paths mint a correlation before invoke and store it on the audit row and any MCP external-input edge.

**Rejected MCP calls do not generate a correlation.** Scope denial (and other rejections that never reach execute) may still write a `rejected` audit row for mutations, but `correlation_id` stays empty—there is no lineage seed and no `resource_links` edge for that call.

## Flow graph retention

`flow.graph` aggregates remaining `resource_links` in a time window (default 7 days; capped by `retention.data_events_days` when set, and at most 90 days). When retention deletes old `data_events`, related detail edges cascade away with them.

**`flow.graph` only covers data within the detail-edge retention period.** Asking for a longer window cannot resurrect expired edges; there is no `flow_edge_daily` (or similar) rollup to extend history past what `resource_links` still holds.

## Decision records

- Live aggregation and ownership: [.agents/notes/implemented/architecture/2026-10-30-lineage-flows-in-analytics.md](../../.agents/notes/implemented/architecture/2026-10-30-lineage-flows-in-analytics.md)
- Window and Trace limits: [.agents/notes/implemented/feature/2026-11-07-flow-graph-live-aggregation.md](../../.agents/notes/implemented/feature/2026-11-07-flow-graph-live-aggregation.md)
- MCP / REST / CLI surface: [.agents/notes/implemented/feature/2026-11-07-lineage-mcp-rest-cli.md](../../.agents/notes/implemented/feature/2026-11-07-lineage-mcp-rest-cli.md)
- Correlation minting (including empty correlation on scope-denied MCP): [.agents/notes/implemented/feature/2026-10-17-data-event-correlation.md](../../.agents/notes/implemented/feature/2026-10-17-data-event-correlation.md)
