# Events Advanced Filtering

**Date**: 2026-06-03
**Status**: draft

## Overview

Enhance the Events page (`/service/web/events`) with four new capabilities: time range selection, full-text search, pipeline-matching filter, and page-number pagination.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Pagination | Offset-based (COUNT + OFFSET/LIMIT) | Direct support for page numbers and totals; event volume unlikely to hit offset-performance ceiling |
| Pipeline filter source | `pipeline_runs` table (pipelines that have actually matched events) | More useful than listing all definitions; avoids showing never-triggered pipelines |
| Time range UI | Alpine.js + native `datetime-local` inputs + shortcut button group | Lightweight, no new vendor dependency, matches existing `btn-group join` pattern |
| Full-text search scope | `source` + `data::text` ILIKE | Covers payload content and origin; does not duplicate the Source/EventType dropdown filters |

## Store Layer

### `ListDataEventsOptions` changes (`internal/store/store.go`)

New fields added to the existing struct:

```go
type ListDataEventsOptions struct {
    Limit        int       // max 100, default 20
    Offset       int       // NEW: page offset for offset-based pagination
    Cursor       string    // kept for backward compatibility; mutually exclusive with Offset
    Source       string    // unchanged
    EventType    string    // unchanged
    Webhook      bool      // unchanged
    Search       string    // NEW: ILIKE match against source and data::text
    PipelineName string    // NEW: filter events that triggered a specific pipeline (join pipeline_runs)
    TimeStart    *time.Time // NEW: created_at >= TimeStart
    TimeEnd      *time.Time // NEW: created_at <= TimeEnd
}
```

### `ListDataEvents` query logic (`store.go:551-593`)

When Offset > 0, use `Offset(offset).Limit(limit)`. When Cursor is set and Offset == 0, use existing cursor logic. The two paths are mutually exclusive.

Additional WHERE clauses:

```go
// Search (source + data payload)
if opts.Search != "" {
    q = q.Where(func(s *sql.Selector) {
        s.Where(sql.ExprP(
            "source ILIKE '%' || $1 || '%' OR data::text ILIKE '%' || $1 || '%'",
            opts.Search,
        ))
    })
}

// Pipeline name (semi-join on pipeline_runs)
if opts.PipelineName != "" {
    q = q.Where(func(s *sql.Selector) {
        t := sql.Table(pipelinerun.Table)
        s.Where(sql.ExprP(
            "event_id IN (SELECT event_id FROM pipeline_runs WHERE pipeline_name = $1)",
            opts.PipelineName,
        ))
    })
}

// Time range
if opts.TimeStart != nil {
    q = q.Where(dataevent.CreatedAtGTE(*opts.TimeStart))
}
if opts.TimeEnd != nil {
    q = q.Where(dataevent.CreatedAtLTE(*opts.TimeEnd))
}
```

### New method: `CountDataEvents`

```go
func (s *EventStore) CountDataEvents(ctx context.Context, opts ListDataEventsOptions) (int64, error)
```

Returns total matching count using the same filter predicates as `ListDataEvents`. Called from the web service to compute total pages.

### New method: `ListDistinctEventPipelineNames`

```go
func (s *EventStore) ListDistinctEventPipelineNames(ctx context.Context) ([]string, error)
```

Queries `pipeline_runs` table: `SELECT DISTINCT pipeline_name FROM pipeline_runs ORDER BY pipeline_name`. Returns only pipelines that have produced at least one `pipeline_run` (i.e., have actually matched events). Used to populate the pipeline filter dropdown.

## Web Service Layer

### File: `internal/modules/web/event_webservice.go`

**New helper: `parseEventFilterParams`**

Parses all filter parameters from `*fiber.Ctx` query string into `ListDataEventsOptions`. Validates time range (end >= start), clamps page to valid range, returns default values for missing params.

Query parameters:

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `source` | string | `""` | Exact match on source |
| `type` | string | `""` | Exact match on event_type |
| `search` | string | `""` | ILIKE search on source + data payload |
| `pipeline` | string | `""` | Pipeline name filter (from pipeline_run matches) |
| `time_start` | string | `""` | RFC3339 start time |
| `time_end` | string | `""` | RFC3339 end time |
| `page` | int | 1 | Current page number |
| `per_page` | int | 20 | Items per page (max 100) |

**Modified handlers:**

- `eventsPage` (GET `/events`): Renders the page shell with filter bar and empty table placeholder. Calls `ListDistinctEventPipelineNames()` to inject pipeline names into the dropdown. No longer preloads event data; the initial data load is triggered via `hx-trigger="load"` on the table partial.
- `filteredEventsTable` (GET `/events/filtered-events`): Merges the old `dataEventsTable` and `webhookLogsTable` into one handler. Accepts `?tab=data-events|webhook-logs` param to toggle `Webhook` filter. Returns the table partial with rows + pagination controls.
- `eventPayload` / webhook payload: No changes.

## Frontend

### New file: `public/js/event-filters.js`

Alpine.js component `eventFilters()` managing filter state:

```js
// Tracks currently selected filter values for HTMX form submission.
// Dropdown option lists (pipeline names, source values, event types)
// are server-rendered in the template, not managed here.
Alpine.data('eventFilters', () => ({
    timeRange: 'custom',   // '1h' | '24h' | '7d' | 'custom'
    timeStart: '',
    timeEnd: '',
    search: '',
    pipeline: '',          // selected value, not the option list
    source: '',            // selected value
    eventType: '',         // selected value
    tab: 'data-events',

    init() {
        // Parse initial values from URL query params
    },

    setTimeRange(range) {
        const now = new Date();
        const durations = { '1h': 3600000, '24h': 86400000, '7d': 604800000 };
        this.timeRange = range;
        if (durations[range]) {
            this.timeEnd = now.toISOString().slice(0, 16);
            this.timeStart = new Date(now - durations[range]).toISOString().slice(0, 16);
        }
        this.submitFilter();
    },

    onDateChange() {
        this.timeRange = 'custom';
        this.submitFilter();
    },

    submitFilter() {
        // Build form params and trigger HTMX request
        // Debounce search input by 300ms
    }
}));
```

### New file: `pkg/views/partials/event_filters.templ`

Filter bar placed above the table. Layout (responsive, stacks on mobile):

```
┌──────────────────────────────────────────────────────────────────────────┐
│ [1h] [24h] [7d]  [datetime-local start] ~ [datetime-local end]           │
│                                                                          │
│ [Search input                     ]  [Pipeline ▼] [Source ▼] [Type ▼]    │
└──────────────────────────────────────────────────────────────────────────┘
```

- Time shortcut buttons: `btn btn-sm join-item`, active state via Alpine `x-bind:class`, sets `timeRange` on click.
- Date inputs: `type="datetime-local"` with Alpine `x-model` and `@change="onDateChange()"`.
- Search input: `type="search"` with `hx-trigger="keyup changed delay:300ms"`.
- Pipeline dropdown: options injected server-side at page render time via `ListDistinctEventPipelineNames()`, same pattern as Source/EventType from `FilterCache`. Default option "All pipelines".
- The Alpine.js component tracks the _selected_ value of each dropdown (for HTMX form submission), not the option list.
- `GET /events/pipeline-names` endpoint exists for potential AJAX refresh but is not used during normal operation.
- Source/EventType dropdowns: populated from `types.EventFilterCache` (unchanged from existing).

### Modified files

**`pkg/views/partials/data_events_table.templ`**

- Remove the internal `<form>` with Source/EventType selects (these move to `event_filters.templ`).
- Replace "Load more" button with pagination control partial.
- Accept `PageInfo` struct: `Page`, `TotalPages`, `Total`, `PerPage`, `HasPrev`, `HasNext`.

**`pkg/views/partials/webhook_logs_table.templ`**

- Same changes as `data_events_table.templ`.

**`pkg/views/partials/event_pagination.templ`** (new)

Pagination control:

```
[Per page: 20 ▼]  Showing 21-40 of 157  [Prev] [1] [2] [3] ... [8] [Next]  Go to: [__] [Go]
```

- Per-page selector: `select` with options 10/20/50/100, changes trigger HTMX reload with `page=1`.
- Page buttons: max 7 visible (first, last, current ± 2, ellipsis for gaps).
- Jump input: small text input + Go button, validates numeric input, clamps to valid range.
- All navigation buttons use HTMX `hx-get` with updated `page` param, targeting `#events-table-container`.

**`pkg/views/pages/events.templ`**

- Wrap table area in `x-data="eventFilters()"` container.
- Include `@event_filters.FilterBar(...)` above the tabs.
- Tab switching preserves filter query params via `hx-push-url="true"`.
- Initial data load via `hx-trigger="load"` on the table partial element.

## Data Flow

```
User action (click shortcut, type search, change select, click page button)
  └─► Alpine.js updates filter state
       └─► HTMX GET /events/filtered-events?tab=...&search=...&pipeline=...&page=...
            └─► parseEventFilterParams(c)
                 └─► EventStore.CountDataEvents(ctx, opts)  ──► total
                 └─► EventStore.ListDataEvents(ctx, opts)    ──► events
                 └─► EventStore.GetPipelineRunsForEvents()   ──► event -> pipeline name map
                      └─► Render: table rows + pagination partial
```

## Backward Compatibility

- `cursor` parameter on GET `/events/data-events` still works; if `cursor` is present, `Offset` is ignored and cursor-based pagination is used.
- `/events/webhook-logs` path is deprecated in favor of `/events/filtered-events?tab=webhook-logs`, but kept as a redirect for external links.
- `types.EventFilterCache` continues to serve Source/EventType dropdown values.

## Edge Cases and Error Handling

| Scenario | Behavior |
|----------|----------|
| Empty search string | Ignore Search condition (no ILIKE added to WHERE) |
| Invalid time range (end < start) | Ignore time filter entirely |
| Page out of bounds | Clamp to last page; if total == 0, show page 1 with empty table |
| Per-page exceeds max (100) | Clamp to 100 |
| Pipeline name not found | Return 0 results (valid but empty) |
| Search with special SQL chars | Parameterized query via `$1`, safe from injection |
| Concurrent filter cache hydration race | `sync.RWMutex` already in FilterCache; no changes needed |

## Testing

### Store tests (`internal/store/store_test.go`)

Extend `TestListDataEvents` with table cases:

- Full-text search matches `source` field
- Full-text search matches `data` payload (JSONB text)
- Full-text search no match
- Pipeline name filter returns only matched events
- Pipeline name not found returns empty
- `TimeStart` range filter
- `TimeEnd` range filter
- Combined `TimeStart` + `TimeEnd`
- Offset-based pagination page 1
- Offset-based pagination page 2
- Offset + limit boundary

New tests:

- `TestCountDataEvents`: 3+ cases verifying count matches filtered List result
- `TestListDistinctEventPipelineNames`: 3+ cases with 0/1/multiple pipeline names

### Web service tests

- `TestParseEventFilterParams`: 3+ cases covering default values, all params set, invalid time range
- `TestFilteredEventsTable`: 3+ cases verifying HTMX response contains pagination controls

### BDD specs (`tests/specs/`)

Extend `event_spec_test.go` with new `Describe` block:

- `Describe("Events advanced filtering", ...)`: scenarios for search, time range, pipeline filter, pagination with page numbers

## Files Changed Summary

| Action | File | Description |
|--------|------|-------------|
| Modify | `internal/store/store.go` | Extended `ListDataEventsOptions`, new `CountDataEvents`, `ListDistinctEventPipelineNames` |
| Modify | `internal/modules/web/event_webservice.go` | New `parseEventFilterParams`, merged table handler |
| New | `public/js/event-filters.js` | Alpine.js `eventFilters()` component |
| New | `pkg/views/partials/event_filters.templ` | Filter bar partial |
| New | `pkg/views/partials/event_pagination.templ` | Pagination control partial |
| Modify | `pkg/views/partials/data_events_table.templ` | Remove internal filter form, add pagination |
| Modify | `pkg/views/partials/webhook_logs_table.templ` | Same as above |
| Modify | `pkg/views/pages/events.templ` | Alpine.js container, include filter bar |
| Modify | `internal/store/store_test.go` | New test cases |
| New | `internal/modules/web/event_webservice_test.go` | Web service tests |
