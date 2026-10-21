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

## Time range

Default **7d**. Presets: **24h** (rolling), **7d**, **30d**, or custom `since`/`until`. The same range query applies across Personal and Ops pages.

## Configuration

```yaml
modules:
  - name: analytics
    enabled: true
```

Decision record: [.agents/notes/implemented/feature/2026-10-21-analytics-replaces-life.md](../../.agents/notes/implemented/feature/2026-10-21-analytics-replaces-life.md).
