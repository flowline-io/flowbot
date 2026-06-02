# Events Page

**Date:** 2026-06-02  
**Status:** Approved  
**Scope:** Web module — new page + store queries + webhook data recording

## Problem

There is no centralized UI to view DataEvent history and webhook receipt logs. When a pipeline fails to trigger, operators must query the database directly to determine whether the event was received, what its payload contained, and which pipelines matched (or should have matched).

## Design

### Page: `/service/web/events`

A new page with two DaisyUI tabs:

1. **Data Events** — All DataEvent records with pipeline matching
2. **Webhook Logs** — Webhook receipt history (filtered from DataEvents)

### Tab: Data Events

**Table columns:**

| Time | Event Type | Source / Capability | Entity | Pipeline |
|------|-----------|---------------------|--------|----------|
| `created_at` | `event_type` | `source` / `capability` | `entity_id` | Matched pipelines |

**Filter bar** (above the table):
- Source dropdown — populated from `SELECT DISTINCT source FROM data_events`
- Event Type dropdown — populated from `SELECT DISTINCT event_type FROM data_events`
- Apply button triggers HTMX reload of the table partial

**Pipeline match column:**
- **Green badge**: pipeline run exists in `pipeline_runs` (actual triggered run)
- **Gray badge**: pipeline definition matches via `FindByEvent` but no run was created (would-be match)
- **"(none)"**: no pipeline definition matched the event type

**Expandable row:** Click a row loads the full event payload via HTMX GET to `/service/web/events/payload/:eventID`. The detail row renders JSON in a `<pre>` block inside `<details open><summary>Payload</summary>`.

**Pagination:** Cursor-based. "Load more" button at bottom appends the next page of rows via HTMX.

### Tab: Webhook Logs

**Table columns:**

| Time | Source | Path | Method | Status | Pipeline |
|------|--------|------|--------|--------|----------|
| `created_at` | `source` (e.g. `github`) | `_webhook_path` | `_webhook_method` | `_webhook_status` (202) | Matched pipelines |

**Filtering:** Only events where `_webhook_method` is set in Data (i.e., events originating from webhook receipts). Same source/event-type dropdowns as DataEvents tab.

**Expandable row:** Click loads request headers + body from `_webhook_headers` and `_webhook_body` in the Data JSONB.

**Pipeline column:** Same logic as DataEvents tab.

### Webhook Data Recording

Two existing webhook entry points must record additional metadata into `DataEvent.Data`:

| Location | Keys to set |
|----------|------------|
| `internal/server/webhook.go:makeWebhookHandler()` | `_webhook_method`, `_webhook_path`, `_webhook_status` |
| `pkg/ability/eventsource.go:WebhookHandler()` | `_webhook_method`, `_webhook_path`, `_webhook_status` on each event from `Convert()` |

Values: `_webhook_method` = HTTP method string (GET/POST), `_webhook_path` = request URL path, `_webhook_status` = `202` (always accepted before async processing).

No new database tables or columns — all data lives in the existing `data_events.data` JSONB column.

### Store Layer

New methods on `EventStore` (`internal/store/store.go`):

```go
type ListDataEventsOptions struct {
    Limit     int
    Cursor    string
    Source    string
    EventType string
    Webhook   bool
}

ListDataEvents(ctx, opts) ([]*ent.DataEvent, nextCursor string, error)
ListDistinctEventSources(ctx) ([]string, error)
ListDistinctEventTypes(ctx) ([]string, error)
GetPipelineRunsForEvents(ctx, eventIDs []string) (map[string][]PipelineRunInfo, error)
```

Cursor pagination uses the limit+1 pattern (matching existing `FindResourcesByTag`).

### Route Table

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| GET | `/service/web/events` | `eventsPage` | Full page (templ) |
| GET | `/service/web/events/data-events` | `dataEventsTable` | HTML partial |
| GET | `/service/web/events/webhook-logs` | `webhookLogsTable` | HTML partial |
| GET | `/service/web/events/payload/:eventID` | `eventPayload` | HTML partial |

### Pipeline Match Computation

Handler logic for each event row:

1. Batch-lookup `pipeline_runs` for displayed event IDs → actual triggered pipelines
2. For events with no runs, call `loader.FindByEvent(defs, eventType)` → would-be matches, where `defs` is obtained from `engine.GetDefinitions()` (cached in-memory)
3. Render a combined list of green + gray badges, or "(none)"

### Authentication

The events page requires an active session (same as configs/pipelines pages). No `route.WithNotAuth()` — this is internal debugging tooling.

## Files Changed

| File | Change |
|------|--------|
| `internal/store/store.go` | Add `ListDataEvents`, `ListDistinctEventSources`, `ListDistinctEventTypes`, `GetPipelineRunsForEvents` |
| `internal/modules/web/event_webservice.go` | New file: route handlers |
| `internal/modules/web/module.go` | Wire `eventWebserviceRules` into `Webservice()` |
| `pkg/views/pages/events.templ` | New file: full page template |
| `pkg/views/partials/data_events_table.templ` | New file: DataEvent table partial |
| `pkg/views/partials/webhook_logs_table.templ` | New file: webhook log table partial |
| `internal/server/webhook.go` | Set `_webhook_method`, `_webhook_path`, `_webhook_status` in DataEvent.Data |
| `pkg/ability/eventsource.go` | Set `_webhook_method`, `_webhook_path`, `_webhook_status` on converted events |

## Testing

**Unit tests** (`event_webservice_test.go`) — table-driven, store mock:

| Test | Cases |
|------|-------|
| `TestListDataEvents` | happy path with results, empty, pagination with cursor |
| `TestListDataEventsFilters` | source filter only, event type filter only, both combined |
| `TestListDataEventsPipelineMatch` | actual run matched, would-be match only, no match |
| `TestListWebhookLogs` | webhook events returned, non-webhook excluded, empty |
| `TestExpandPayload` | valid JSON, empty payload, missing event |

**Store tests** (`store_test.go`) — extend existing:

| Test | Cases |
|------|-------|
| `TestListDataEventsPagination` | page 1 returns N, last page returns <N, cursor yields next page |
| `TestListDataEventsFilterBySource` | known source, unknown source, empty source (all) |
| `TestListDistinctSources` | multiple sources, single source, no events |

**BDD specs** — Ginkgo v2: full page renders both tabs, filter dropdowns populate, clicking rows expands payload, pagination works.

## Out of Scope

- Real-time SSE updates for new events (page requires manual reload or HTMX refresh)
- Date range filter (can be added later)
- Bulk event deletion or purging
- Webhook retry/replay functionality
