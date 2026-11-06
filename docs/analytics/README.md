# Analytics

Flowbot's personal usage and ops run-health surface under User nav.

| Layer | Location |
| ----- | -------- |
| Module | `internal/modules/analytics` |
| Web | `/service/web/analytics*` |
| Aggregation | Existing domain stores (`LLMUsageStore`, `EventStore`, `PipelineStore`, `FunctionStore`, …) — live queries, no rollup tables |

## Personal (default)

- Overview — token / events / owned runs / ops success headline
- Token usage — `/service/web/analytics/token-usage` (`/home/token-usage` redirects here)
- Events — `data_events` for the signed-in `uid`, groupBy `event_type` \| `capability` \| `source`
- Activity — pipeline and function runs for definitions with `created_by` = current user. Workflows are omitted: workflow definitions have no ownership field.

## Ops

- Runs health — site-wide pipeline / workflow / function success rate trends, failure counts, and duration distributions

## Flows

Third Analytics partition (Personal / Ops / Flows). Web: `/service/web/analytics/flows` — ECharts topology (default) and Sankey toggle; failed edges red; UI caps at 20 nodes / 50 edges via `FlowGraphUI`. Live `flow.graph` / multi-hop `Trace` also expose MCP, REST (`/service/lineage|flows|events`), and CLI over `resource_links` (no daily rollup table). Decision: [.agents/notes/implemented/architecture/2026-10-30-lineage-flows-in-analytics.md](../../.agents/notes/implemented/architecture/2026-10-30-lineage-flows-in-analytics.md); aggregation: [.agents/notes/implemented/feature/2026-11-07-flow-graph-live-aggregation.md](../../.agents/notes/implemented/feature/2026-11-07-flow-graph-live-aggregation.md); UI: [.agents/notes/implemented/feature/2026-11-07-analytics-flows-ui.md](../../.agents/notes/implemented/feature/2026-11-07-analytics-flows-ui.md).

## Time range

Default **7d**. Presets: **24h** (rolling), **7d**, **30d**, or custom `since`/`until`. The same range query applies across Personal, Ops, and Flows pages. `flow.graph` also defaults to 7d and caps the window by `retention.data_events_days` (when set) and at most 90 days.

## Configuration

```yaml
modules:
  - name: analytics
    enabled: true
```

Decision record: [.agents/notes/implemented/feature/2026-10-21-analytics-replaces-life.md](../../.agents/notes/implemented/feature/2026-10-21-analytics-replaces-life.md).
