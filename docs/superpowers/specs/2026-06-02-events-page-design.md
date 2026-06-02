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
- Source dropdown — populated from an in-memory cache updated on each event write (see Performance section)
- Event Type dropdown — same cache strategy
- Apply button triggers HTMX reload of the table partial with `hx-push-url="true"` to persist filter state in the URL query string

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

All data lives in the existing `data_events.data` JSONB column. The following keys are stored:

| Key | Source | Description |
|-----|--------|-------------|
| `_webhook_method` | New — add to both handlers | HTTP method (`GET`/`POST`) |
| `_webhook_path` | New — add to both handlers | Request URL path |
| `_webhook_status` | New — add to both handlers | HTTP response status (`202`) |
| `_webhook_headers` | Already stored by `makeWebhookHandler()` | Sanitized request headers (JSON object). Provider webhooks must also store this. |
| `_webhook_body` | Already stored by `makeWebhookHandler()` (raw mode) | Raw request body string. Provider webhooks must also store this. |

Two existing webhook entry points need changes:

**`internal/server/webhook.go:makeWebhookHandler()`** — already stores `_webhook_body` and `_webhook_headers`. Add `_webhook_method`, `_webhook_path`, `_webhook_status`.

**`pkg/ability/eventsource.go:WebhookHandler()`** — currently stores nothing in Data. After calling `Convert()`, set all five keys on each returned `DataEvent`. Read request body from `ctx.Body()`, headers from `ctx.GetReqHeaders()`, method from `ctx.Method()`, path from `ctx.Path()`.

**Body truncation:** Webhook bodies may be large (multi-MB POST payloads). Truncate `_webhook_body` to 64KB. If truncation occurred, set `_webhook_body_truncated: true` in Data. The expandable detail row shows the truncated body with a visible "(truncated to 64KB)" label.

**Header sanitization:** The existing `sanitizeWebhookHeaders()` function in `webhook.go` already strips `Authorization`, `Cookie`, `X-Api-Key`, HMAC signatures, and configured auth headers before storing `_webhook_headers`. The same sanitizer must be called in `eventsource.go` as well.

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
ListDistinctEventSources(ctx, since time.Duration) ([]string, error)
ListDistinctEventTypes(ctx, since time.Duration) ([]string, error)
GetPipelineRunsForEvents(ctx, eventIDs []string) (map[string][]PipelineRunInfo, error)
```

Cursor pagination uses the limit+1 pattern (matching existing `FindResourcesByTag`).

`since` bounds the distinct queries to recent data (default 30 days) via `WHERE created_at > now() - since`. This prevents full-table scans on the `data_events` table as it grows to millions of rows.

### Performance: Filter Dropdown Caching

`SELECT DISTINCT source/event_type` on a large `data_events` table is expensive. Instead:

1. A **filter cache** (`pkg/module/filter_cache.go`) maintains two in-memory `[]string` slices for sources and event types
2. On module startup, the cache hydrates from `ListDistinctEventSources(ctx, 30*24*time.Hour)` and `ListDistinctEventTypes(ctx, 30*24*time.Hour)`
3. On every `AppendDataEvent()` call, `SetEventSource(event.Source)` and `SetEventType(event.EventType)` push unseen values into the cache (async, non-blocking)
4. The handler reads from the cache to populate dropdowns — no database query on page load

### Performance: Database Indexes

Three indexes needed for efficient querying:

```sql
-- Webhook Logs tab: partial index filtering only webhook events
CREATE INDEX idx_data_events_webhook ON data_events (created_at DESC, id DESC)
WHERE data->>'_webhook_method' IS NOT NULL;

-- General cursor pagination
CREATE INDEX idx_data_events_pagination ON data_events (created_at DESC, id DESC);

-- Filtered pagination by source
CREATE INDEX idx_data_events_source_time ON data_events (source, created_at DESC, id DESC);
```

These are created via ent migration hooks, same as existing indexes.

### Route Table

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| GET | `/service/web/events` | `eventsPage` | Full page (templ) |
| GET | `/service/web/events/data-events` | `dataEventsTable` | HTML partial |
| GET | `/service/web/events/webhook-logs` | `webhookLogsTable` | HTML partial |
| GET | `/service/web/events/payload/:eventID` | `eventPayload` | HTML partial |

### Pipeline Match Computation

This handles the case where one event hits multiple pipeline definitions, but only some actually fire (others may be disabled or skipped by conditions).

Handler logic for the current page of events:

1. Batch-lookup `pipeline_runs` for displayed event IDs → map of `eventID → []pipelineName` (actual triggered pipelines)
2. For every displayed event, call `loader.FindByEvent(defs, event.EventType)` → map of `eventID → []pipelineName` (theoretical matches), where `defs` is obtained from `engine.GetDefinitions()` (cached in-memory)
3. For each event, compute the intersection:
   - **Green badge**: pipeline name appears in both actual runs AND theoretical matches (triggered)
   - **Gray badge**: pipeline name appears in theoretical matches but NOT in actual runs (would-be match, skipped/disabled)
   - **"(none)"**: theoretical matches is empty (no pipeline definition cares about this event type)

An event that triggers Pipeline A (has run row) and also matches Pipeline B in theory (but B is disabled) would show both a green "A" and a gray "B".

### Authentication & Security

**RBAC:** The events page and all its sub-routes require active session + admin role. A new `route.WithRole("admin")` middleware gates access. Event payloads and webhook bodies may contain sensitive data (API tokens, PII) and must not be visible to unprivileged users.

**XSS defense:** All JSON payload rendering in `<pre>` blocks goes through templ's auto-escaping, which escapes HTML characters by default. No manual string concatenation or `unsafe` usage in payload rendering.

**Header sanitization:** The `_webhook_headers` sanitizer already strips `Authorization`, `Cookie`, `X-Api-Key`, and HMAC signature headers. The same function is applied in both webhook entry points.

**URL state:** All HTMX tab switches and filter applications use `hx-push-url="true"` so that the browser URL reflects the current state (e.g., `/service/web/events?tab=data-events&source=github&type=issue_created`). Page reload or link sharing preserves the view.

## Files Changed

| File | Change |
|------|--------|
| `internal/store/store.go` | Add `ListDataEvents`, `ListDistinctEventSources`, `ListDistinctEventTypes`, `GetPipelineRunsForEvents`; update `AppendDataEvent` to notify filter cache |
| `pkg/module/filter_cache.go` | New file: in-memory source/event-type cache for dropdowns |
| `internal/modules/web/event_webservice.go` | New file: route handlers |
| `internal/modules/web/module.go` | Wire `eventWebserviceRules` into `Webservice()`; init filter cache on startup |
| `pkg/views/pages/events.templ` | New file: full page template |
| `pkg/views/partials/data_events_table.templ` | New file: DataEvent table partial |
| `pkg/views/partials/webhook_logs_table.templ` | New file: webhook log table partial |
| `internal/server/webhook.go` | Set `_webhook_method`, `_webhook_path`, `_webhook_status` in DataEvent.Data |
| `pkg/ability/eventsource.go` | Store headers, body, method, path, status on converted events; apply header sanitization |
| `internal/store/ent/schema/` | Migration: add composite indexes for `data_events` table |

## Testing

**Unit tests** (`event_webservice_test.go`) — table-driven, store mock:

| Test | Cases |
|------|-------|
| `TestListDataEvents` | happy path with results, empty, pagination with cursor |
| `TestListDataEventsFilters` | source filter only, event type filter only, both combined |
| `TestListDataEventsPipelineMatch` | actual run matched only, would-be match only, both actual+would-be for same event, no match |
| `TestListWebhookLogs` | webhook events returned, non-webhook excluded, empty |
| `TestExpandPayload` | valid JSON, empty payload, missing event |
| `TestAdminRoleRequired` | unauthenticated returns 401, non-admin session returns 403, admin session succeeds |

**Filter cache tests** (`pkg/module/filter_cache_test.go`):

| Test | Cases |
|------|-------|
| `TestFilterCacheAddSource` | new source added, duplicate ignored, concurrent writes |
| `TestFilterCacheHydrate` | empty initial state, hydrate from DB data, dedup with existing |
| `TestFilterCacheList` | single source, multiple sources, empty cache |

**Store tests** (`store_test.go`) — extend existing:

| Test | Cases |
|------|-------|
| `TestListDataEventsPagination` | page 1 returns N, last page returns <N, cursor yields next page |
| `TestListDataEventsFilterBySource` | known source, unknown source, empty source (all) |
| `TestListDistinctSources` | multiple sources, single source, no events, time-bounded (old events excluded) |

**BDD specs** — Ginkgo v2: full page renders both tabs, filter dropdowns populate from cache, clicking rows expands payload, pagination works, RBAC gates admin-only access, URL state persists across tab/filter changes.

## Out of Scope

- Real-time SSE updates for new events (page requires manual reload or HTMX refresh)
- Date range filter (can be added later)
- Bulk event deletion or purging
- Webhook retry/replay functionality
