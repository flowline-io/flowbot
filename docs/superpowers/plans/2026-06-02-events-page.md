# Events Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `/service/web/events` page with two tabs (Data Events + Webhook Logs) for debugging event flows and pipeline matching.

**Architecture:** Store queries add paginated event listing with filters. A filter cache eliminates expensive SELECT DISTINCT on large tables. Webhook entry points record HTTP method/path/status into DataEvent.Data JSONB. templ templates render DaisyUI tables with HTMX expandable rows and hx-push-url state persistence. Auth gates access to admin-scoped sessions.

**Tech Stack:** Go 1.26+, ent (PostgreSQL), Fiber v3, templ v0.3, DaisyUI v5, HTMX 2.x

**Target files: 10 files created, 5 files modified**

| Op | File |
|----|------|
| Modify | `internal/store/store.go` |
| Create | `pkg/types/filter_cache.go` |
| Create | `pkg/types/filter_cache_test.go` |
| Create | `internal/modules/web/event_webservice.go` |
| Modify | `internal/modules/web/module.go` |
| Create | `pkg/views/pages/events.templ` |
| Create | `pkg/views/partials/data_events_table.templ` |
| Modify | `pkg/views/layout/base.templ` |
| Modify | `internal/server/webhook.go` |
| Modify | `pkg/ability/eventsource.go` |
| Create | `internal/store/store_test.go` (extend: add new test funcs) |
| Create | `internal/modules/web/event_webservice_test.go` |
| Create | `pkg/views/partials/event_payload.templ` |
| Create | `pkg/views/partials/webhook_payload.templ` |

**Dependencies:** Tasks 1-5 (store + cache) are independent of Tasks 6-7 (webhook recording), which are independent of Tasks 8-12 (templates + handlers). Execute store tasks first, then the rest in parallel.

---

### Task 1: Add `ListDataEvents` store method

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add `ListDataEventsOptions` and `ListDataEvents` method**

Add after the `MarkOutboxPublished` method (after line 522):

```go
// ListDataEventsOptions holds filters and pagination for listing data events.
type ListDataEventsOptions struct {
	Limit     int    // max 100, default 20
	Cursor    string // opaque CreatedAt cursor
	Source    string // filter by source, empty = all
	EventType string // filter by event type, empty = all
	Webhook   bool   // if true, only events where data->>'_webhook_method' IS NOT NULL
}

// ListDataEvents returns paginated data_events ordered by created_at DESC.
// Uses cursor-based pagination (limit+1 pattern).
func (s *EventStore) ListDataEvents(ctx context.Context, opts ListDataEventsOptions) ([]*gen.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := s.client.DataEvent.Query().
		Order(dataevent.ByCreatedAt(sql.OrderDesc())).
		Limit(opts.Limit + 1)

	if opts.Source != "" {
		q = q.Where(dataevent.Source(opts.Source))
	}
	if opts.EventType != "" {
		q = q.Where(dataevent.EventType(opts.EventType))
	}
	if opts.Webhook {
		q = q.Where(func(selector *sql.Selector) {
			selector.Where(sql.ExprP("data->>'_webhook_method' IS NOT NULL"))
		})
	}

	if opts.Cursor != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999Z", opts.Cursor); err == nil {
			q = q.Where(dataevent.CreatedAtLT(t))
		}
	}

	events, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("list data events: %w", err)
	}

	var nextCursor string
	if len(events) > opts.Limit {
		nextCursor = events[opts.Limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
		events = events[:opts.Limit]
	}

	return events, nextCursor, nil
}
```

- [ ] **Step 2: Add imports used by the new code**

Ensure these imports are present in `store.go`:
- `"fmt"` (already present)
- `"time"` (already present)
- `dataevent` from ent gen (already imported)
- `"entgo.io/ent/dialect/sql"` (check if already imported; used for `sql.ExprP` and `sql.OrderDesc`)

Run: `go vet ./internal/store/`
Expected: no errors.

---

### Task 2: Add `ListDistinctEventSources`, `ListDistinctEventTypes` and `GetDataEventByEventID` store methods

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add `ListDistinctEventSources` method**

Add after `ListDataEvents`:

```go
// ListDistinctEventSources returns unique source values from data_events
// created within the given duration (e.g. 30*24*time.Hour for last 30 days).
func (s *EventStore) ListDistinctEventSources(ctx context.Context, since time.Duration) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	var sources []string
	err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtGT(time.Now().Add(-since))).
		GroupBy(dataevent.FieldSource).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct event sources: %w", err)
	}
	return sources, nil
}
```

- [ ] **Step 2: Add `ListDistinctEventTypes` method**

```go
// ListDistinctEventTypes returns unique event_type values from data_events
// created within the given duration.
func (s *EventStore) ListDistinctEventTypes(ctx context.Context, since time.Duration) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	var types []string
	err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtGT(time.Now().Add(-since))).
		GroupBy(dataevent.FieldEventType).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct event types: %w", err)
	}
	return types, nil
}
```

- [ ] **Step 3: Add `GetDataEventByEventID` method**

```go
// GetDataEventByEventID looks up a single data event by its event_id.
func (s *EventStore) GetDataEventByEventID(ctx context.Context, eventID string) (*gen.DataEvent, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	e, err := s.client.DataEvent.Query().
		Where(dataevent.EventID(eventID)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get data event by id: %w", err)
	}
	return e, nil
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/store/`
Expected: no errors.

---

### Task 3: Add `GetPipelineRunsForEvents` store method

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add `PipelineRunInfo` type and `GetPipelineRunsForEvents` method**

Add before the EventStore struct:

```go
// PipelineRunInfo is a lightweight view of a pipeline run for event matching display.
type PipelineRunInfo struct {
	PipelineName string
	EventID      string
	Status       string
}
```

Add method to `EventStore`:

```go
// GetPipelineRunsForEvents batch-looks up pipeline runs for the given event IDs.
// Returns a map of eventID -> []PipelineRunInfo.
func (s *EventStore) GetPipelineRunsForEvents(ctx context.Context, eventIDs []string) (map[string][]PipelineRunInfo, error) {
	if s == nil || s.client == nil || len(eventIDs) == 0 {
		return nil, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.EventIDIn(eventIDs...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pipeline runs for events: %w", err)
	}
	result := make(map[string][]PipelineRunInfo, len(runs))
	for _, r := range runs {
		info := PipelineRunInfo{
			PipelineName: r.PipelineName,
			EventID:      r.EventID,
			Status:       r.Status.String(),
		}
		result[r.EventID] = append(result[r.EventID], info)
	}
	return result, nil
}
```

- [ ] **Step 2: Add import for `pipelinerun`**

Ensure `pipelinerun` from ent gen is imported.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/store/`
Expected: no errors.

---

### Task 4: Write store unit tests for ListDataEvents

**Files:**
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Add `TestListDataEvents`**

```go
func TestListDataEvents(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewEventStore(client)
	ctx := context.Background()

	now := time.Now()
	events := []*gen.DataEvent{
		{EventID: "evt-001", EventType: "issue.created", Source: "github", Capability: "forge", EntityID: "repo#42", CreatedAt: now},
		{EventID: "evt-002", EventType: "bookmark.created", Source: "karakeep", Capability: "bookmark", EntityID: "url-1", CreatedAt: now.Add(-1 * time.Hour)},
		{EventID: "evt-003", EventType: "entry.new", Source: "reader", Capability: "reader", EntityID: "feed-5", CreatedAt: now.Add(-2 * time.Hour)},
	}

	for _, e := range events {
		de := types.DataEvent{
			EventID:    e.EventID,
			EventType:  e.EventType,
			Source:     e.Source,
			Capability: e.Capability,
			EntityID:   e.EntityID,
			CreatedAt:  e.CreatedAt,
		}
		require.NoError(t, store.AppendDataEvent(ctx, de))
	}

	tests := []struct {
		name          string
		opts          ListDataEventsOptions
		wantCount     int
		wantHasCursor bool
	}{
		{
			name:      "list all events",
			opts:      ListDataEventsOptions{Limit: 10},
			wantCount: 3,
		},
		{
			name:      "filter by source github",
			opts:      ListDataEventsOptions{Limit: 10, Source: "github"},
			wantCount: 1,
		},
		{
			name:      "filter by unknown source returns empty",
			opts:      ListDataEventsOptions{Limit: 10, Source: "unknown"},
			wantCount: 0,
		},
		{
			name:      "filter by event type",
			opts:      ListDataEventsOptions{Limit: 10, EventType: "bookmark.created"},
			wantCount: 1,
		},
		{
			name:      "pagination with cursor returns cursor",
			opts:      ListDataEventsOptions{Limit: 1},
			wantCount: 1,
			wantHasCursor: true,
		},
		{
			name:      "pagination last page no cursor",
			opts:      ListDataEventsOptions{Limit: 10},
			wantCount: 3,
			wantHasCursor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, cursor, err := store.ListDataEvents(ctx, tt.opts)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)
			if tt.wantHasCursor {
				assert.NotEmpty(t, cursor)
			} else {
				assert.Empty(t, cursor)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/store/ -run TestListDataEvents -v -count=1`
Expected: PASS.

---

### Task 5: Create filter cache

**Files:**
- Create: `pkg/types/filter_cache.go`
- Create: `pkg/types/filter_cache_test.go`

- [ ] **Step 1: Write filter cache implementation**

```go
package types

import "sync"

// FilterCache holds in-memory unique sets of sources and event types
// used to populate filter dropdowns without querying the database.
type FilterCache struct {
	mu         sync.RWMutex
	sources    []string
	eventTypes []string
	sourceSet  map[string]struct{}
	typeSet    map[string]struct{}
}

// NewFilterCache creates an empty FilterCache.
func NewFilterCache() *FilterCache {
	return &FilterCache{
		sourceSet: make(map[string]struct{}),
		typeSet:   make(map[string]struct{}),
	}
}

// SetSource adds a source to the cache if not already present.
func (fc *FilterCache) SetSource(source string) {
	if source == "" {
		return
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if _, ok := fc.sourceSet[source]; ok {
		return
	}
	fc.sourceSet[source] = struct{}{}
	fc.sources = append(fc.sources, source)
}

// SetEventType adds an event type to the cache if not already present.
func (fc *FilterCache) SetEventType(eventType string) {
	if eventType == "" {
		return
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if _, ok := fc.typeSet[eventType]; ok {
		return
	}
	fc.typeSet[eventType] = struct{}{}
	fc.eventTypes = append(fc.eventTypes, eventType)
}

// Sources returns a copy of all cached sources.
func (fc *FilterCache) Sources() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	srcs := make([]string, len(fc.sources))
	copy(srcs, fc.sources)
	return srcs
}

// EventTypes returns a copy of all cached event types.
func (fc *FilterCache) EventTypes() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	types := make([]string, len(fc.eventTypes))
	copy(types, fc.eventTypes)
	return types
}

// Hydrate populates the cache from database lists (deduplicates with existing).
func (fc *FilterCache) Hydrate(sources, eventTypes []string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	for _, s := range sources {
		if _, ok := fc.sourceSet[s]; !ok {
			fc.sourceSet[s] = struct{}{}
			fc.sources = append(fc.sources, s)
		}
	}
	for _, t := range eventTypes {
		if _, ok := fc.typeSet[t]; !ok {
			fc.typeSet[t] = struct{}{}
			fc.eventTypes = append(fc.eventTypes, t)
		}
	}
}
```

- [ ] **Step 2: Write filter cache tests**

```go
package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterCacheSetSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		sources   []string
		wantCount int
	}{
		{
			name:      "add single source",
			sources:   []string{"github"},
			wantCount: 1,
		},
		{
			name:      "add duplicate source ignored",
			sources:   []string{"github", "github"},
			wantCount: 1,
		},
		{
			name:      "add multiple distinct sources",
			sources:   []string{"github", "gitea", "reader"},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFilterCache()
			for _, s := range tt.sources {
				fc.SetSource(s)
			}
			assert.Len(t, fc.Sources(), tt.wantCount)
		})
	}
}

func TestFilterCacheHydrate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		initialSrcs   []string
		hydrateSrcs   []string
		wantSrcCount  int
	}{
		{
			name:         "hydrate into empty cache",
			initialSrcs:  nil,
			hydrateSrcs:  []string{"github", "gitea"},
			wantSrcCount: 2,
		},
		{
			name:         "hydrate with overlap deduplicates",
			initialSrcs:  []string{"github"},
			hydrateSrcs:  []string{"github", "reader"},
			wantSrcCount: 2,
		},
		{
			name:         "hydrate empty list preserves existing",
			initialSrcs:  []string{"github"},
			hydrateSrcs:  nil,
			wantSrcCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFilterCache()
			for _, s := range tt.initialSrcs {
				fc.SetSource(s)
			}
			fc.Hydrate(tt.hydrateSrcs, nil)
			assert.Len(t, fc.Sources(), tt.wantSrcCount)
		})
	}
}

func TestFilterCacheEmptySource(t *testing.T) {
	t.Parallel()
	fc := NewFilterCache()
	fc.SetSource("")
	assert.Empty(t, fc.Sources())
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/types/ -run TestFilterCache -v -count=1`
Expected: PASS.

---

### Task 6: Wire filter cache into store's AppendDataEvent

**Files:**
- Modify: `internal/store/store.go`
- Modify: `pkg/types/filter_cache.go`

- [ ] **Step 1: Add global filter cache in types package**

In `pkg/types/filter_cache.go`, add at the bottom:

```go
// EventFilterCache is the global filter cache for event sources and types.
// Initialized by the web module on startup, updated by the store on event write.
var EventFilterCache = NewFilterCache()
```

- [ ] **Step 2: Update `AppendDataEvent` to notify filter cache**

In `internal/store/store.go`, at the end of `AppendDataEvent` (just before `return err`), add:

```go
	if err == nil && event.Source != "" {
		types.EventFilterCache.SetSource(event.Source)
	}
	if err == nil && event.EventType != "" {
		types.EventFilterCache.SetEventType(event.EventType)
	}
```

- [ ] **Step 3: Verify compilation**

`pkg/types` does not import `internal/store`, so no import cycle. Verify:

Run: `go build ./internal/store/`
Expected: no errors.

---

### Task 7: Record webhook metadata in eventsource.go

**Files:**
- Modify: `pkg/ability/eventsource.go`

- [ ] **Step 1: Add header sanitization function**

Add at the bottom of `eventsource.go`:

```go
// sanitizeEventSourceHeaders removes sensitive headers from the request headers map.
var eventSourceSensitiveHeaders = map[string]bool{
	"Authorization":               true,
	"Cookie":                      true,
	"Set-Cookie":                  true,
	"X-Api-Key":                   true,
	"X-Hub-Signature":             true,
	"X-Hub-Signature-256":         true,
	"X-Hmac-Signature":            true,
	"X-Webhook-Token":             true,
	"X-Gitlab-Token":              true,
	"X-Gogs-Signature":            true,
	"X-Hub-Signature":             true,
}

func sanitizeEventSourceHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if eventSourceSensitiveHeaders[k] {
			continue
		}
		out[k] = v
	}
	return out
}
```

- [ ] **Step 2: In `WebhookHandler()`, after the `convert` call and before submitting events, add webhook metadata to each DataEvent**

In the `WebhookHandler()` method, after `events, err := converter.Convert(body, headers)` and before the `for _, rawEvent := range events` loop, add:

```go
		sanitizedHeaders := sanitizeEventSourceHeaders(headers)
		method := string(c.Request().Header.Method())
		path := string(c.Request().URI().Path())
		if events[i].Data == nil {
			events[i].Data = make(types.KV)
		}
		events[i].Data["_webhook_method"] = method
		events[i].Data["_webhook_path"] = path
		events[i].Data["_webhook_status"] = 202
		events[i].Data["_webhook_headers"] = sanitizedHeaders
		events[i].Data["_webhook_body"] = truncateBody(body)
```

- [ ] **Step 3: Add `truncateBody` helper**

```go
const maxWebhookBodySize = 64 * 1024 // 64KB

func truncateBody(body []byte) string {
	if len(body) <= maxWebhookBodySize {
		return string(body)
	}
	return string(body[:maxWebhookBodySize])
}
```

- [ ] **Step 4: Also set `_webhook_body_truncated` flag when truncation occurs**

Modify step 2: after setting `_webhook_body`, add:

```go
		if len(body) > maxWebhookBodySize {
			events[i].Data["_webhook_body_truncated"] = true
		}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./pkg/ability/`
Expected: no errors.

---

### Task 8: Record webhook metadata in webhook.go

**Files:**
- Modify: `internal/server/webhook.go`

- [ ] **Step 1: In `makeWebhookHandler()`, add method/path/status to DataEvent.Data**

Find the block where `dataEvent.Data` is populated (after `_webhook_body` and `_webhook_headers` are set). Add:

```go
	if dataEvent.Data == nil {
		dataEvent.Data = make(types.KV)
	}
	dataEvent.Data["_webhook_method"] = string(c.Request().Header.Method())
	dataEvent.Data["_webhook_path"] = string(c.Request().URI().Path())
	dataEvent.Data["_webhook_status"] = fiber.StatusAccepted
```

- [ ] **Step 2: Add body truncation for webhook body**

Modify the existing `_webhook_body` assignment in raw mode to truncate:

```go
		body := c.Body()
		dataEvent.Data["_webhook_body"] = truncateWebhookBody(body)
		if len(body) > maxWebhookBodySize {
			dataEvent.Data["_webhook_body_truncated"] = true
		}
```

Add the helper at `webhook.go` bottom or a new file `internal/server/webhook_helpers.go`:

```go
const maxWebhookBodySize = 64 * 1024

func truncateWebhookBody(body []byte) string {
	if len(body) <= maxWebhookBodySize {
		return string(body)
	}
	return string(body[:maxWebhookBodySize])
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/server/`
Expected: no errors.

---

### Task 9: Create events page template

**Files:**
- Create: `pkg/views/pages/events.templ`

- [ ] **Step 1: Write `events.templ`**

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

type EventsPageParams struct {
	ActiveTab     string // "data-events" or "webhook-logs"
	Sources       []string
	EventTypes    []string
	SourceFilter  string
	TypeFilter    string
	Events        interface{} // passed to DataEventsTable or WebhookLogsTable
	NextCursor    string
}

templ EventsPage(p EventsPageParams) {
	@layout.Base("Events") {
		<div class="container mx-auto p-4">
			<h1 class="text-2xl font-bold mb-4">Events</h1>
			<div role="tablist" class="tabs tabs-lifted mb-4">
				<a role="tab"
					class={ "tab", templ.KV("tab-active", p.ActiveTab == "data-events") }
					hx-get={ templ.URL("/service/web/events/data-events?source=" + p.SourceFilter + "&type=" + p.TypeFilter) }
					hx-target="#events-content"
					hx-swap="innerHTML"
					hx-push-url="true"
					hx-trigger="click"
					data-testid="tab-data-events">Data Events</a>
				<a role="tab"
					class={ "tab", templ.KV("tab-active", p.ActiveTab == "webhook-logs") }
					hx-get={ templ.URL("/service/web/events/webhook-logs?source=" + p.SourceFilter + "&type=" + p.TypeFilter) }
					hx-target="#events-content"
					hx-swap="innerHTML"
					hx-push-url="true"
					hx-trigger="click"
					data-testid="tab-webhook-logs">Webhook Logs</a>
			</div>
			<div id="events-content">
				if p.ActiveTab == "webhook-logs" {
					@WebhookLogsTable(p.Sources, p.EventTypes, p.SourceFilter, p.TypeFilter, p.Events, p.NextCursor)
				} else {
					@DataEventsTable(p.Sources, p.EventTypes, p.SourceFilter, p.TypeFilter, p.Events, p.NextCursor)
				}
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Verify templ compilation**

Run: `templ generate pkg/views/pages/events.templ`
Expected: no errors.

---

### Task 10: Create DataEvents table partial

**Files:**
- Create: `pkg/views/partials/data_events_table.templ`

- [ ] **Step 1: Write `data_events_table.templ`**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/gen"

templ DataEventsTable(sources, eventTypes []string, sourceFilter, typeFilter string, events []*gen.DataEvent, nextCursor string) {
	<div class="card bg-base-100 shadow-sm" id="data-events-table" data-testid="data-events-table">
		<!-- Filter bar -->
		<form class="flex flex-wrap items-end gap-2 p-3 bg-base-200 rounded-t-box"
			hx-get="/service/web/events/data-events"
			hx-target="#data-events-table"
			hx-swap="outerHTML"
			hx-push-url="true"
			hx-trigger="submit">
			<label class="form-control max-w-xs">
				<div class="label"><span class="label-text text-xs">Source</span></div>
				<select name="source" class="select select-bordered select-sm">
					<option value="">All</option>
					for _, s := range sources {
						<option value={ s } selected={ templ.KV(s == sourceFilter, true) }>{ s }</option>
					}
				</select>
			</label>
			<label class="form-control max-w-xs">
				<div class="label"><span class="label-text text-xs">Event Type</span></div>
				<select name="type" class="select select-bordered select-sm">
					<option value="">All</option>
					for _, t := range eventTypes {
						<option value={ t } selected={ templ.KV(t == typeFilter, true) }>{ t }</option>
					}
				</select>
			</label>
			<button type="submit" class="btn btn-sm btn-primary" data-testid="filter-apply">Apply</button>
		</form>

		<div class="overflow-x-auto">
			<table class="table">
				<thead>
				<tr>
					<th class="text-xs uppercase">Time</th>
					<th class="text-xs uppercase">Event Type</th>
					<th class="text-xs uppercase">Source / Capability</th>
					<th class="text-xs uppercase">Entity</th>
					<th class="text-xs uppercase">Pipeline</th>
				</tr>
				</thead>
				<tbody id="events-rows">
				for _, e := range events {
					<tbody id={ "event-" + e.EventID }>
						<tr class="cursor-pointer hover"
							hx-get={ templ.URL("/service/web/events/payload/" + e.EventID) }
							hx-trigger="click"
							hx-target={ "#detail-" + e.EventID }
							hx-swap="innerHTML show:top">
							<td class="text-xs whitespace-nowrap">{ e.CreatedAt.Format("15:04:05") }</td>
							<td><span class="badge badge-ghost badge-xs">{ e.EventType }</span></td>
							<td class="text-xs">{ e.Source } / { e.Capability }</td>
							<td class="text-xs font-mono">{ e.EntityID }</td>
							<td id={ "match-" + e.EventID } class="text-xs">
								<span class="loading loading-spinner loading-xs htmx-indicator"></span>
								<span class="text-base-content/50">loading...</span>
							</td>
						</tr>
						<tr id={ "detail-" + e.EventID } class="hidden">
							<td colspan="5"></td>
						</tr>
					</tbody>
				}
				if len(events) == 0 {
					<tr id="events-empty">
						<td colspan="5" class="text-center text-base-content/50 py-4">No events found.</td>
					</tr>
				}
				</tbody>
			</table>
		</div>

		if nextCursor != "" {
			<div class="p-3 text-center" id="load-more-container">
				<button class="btn btn-sm btn-ghost"
					hx-get={ templ.URL("/service/web/events/data-events?source=" + sourceFilter + "&type=" + typeFilter + "&cursor=" + nextCursor) }
					hx-target="#load-more-container"
					hx-swap="outerHTML"
					data-testid="load-more">Load more</button>
			</div>
		}
	</div>
}
```

- [ ] **Step 2: Verify compilation**

Run: `templ generate pkg/views/partials/data_events_table.templ`
Expected: no errors.

---

### Task 11: Create event payload and webhook payload partials

**Files:**
- Create: `pkg/views/partials/event_payload.templ`
- Create: `pkg/views/partials/webhook_payload.templ`

- [ ] **Step 1: Write `event_payload.templ`**

```templ
package partials

templ EventPayloadDetail(payloadJSON string, truncated bool) {
	<td colspan="5" class="p-0">
		<div class="p-3">
			<details open>
				<summary class="text-sm font-semibold cursor-pointer">Payload</summary>
				<pre class="bg-base-200 rounded p-2 text-xs font-mono overflow-x-auto max-h-96 mt-1">{ payloadJSON }</pre>
			</details>
			if truncated {
				<div class="text-xs text-warning mt-1">Payload truncated to 64KB</div>
			}
		</div>
	</td>
}
```

- [ ] **Step 2: Write `webhook_payload.templ`**

```templ
package partials

templ WebhookPayloadDetail(headersJSON, bodyJSON string, bodyTruncated bool) {
	<td colspan="5" class="p-0">
		<div class="p-3 space-y-3">
			<details open>
				<summary class="text-sm font-semibold cursor-pointer">Request Headers</summary>
				<pre class="bg-base-200 rounded p-2 text-xs font-mono overflow-x-auto max-h-48 mt-1">{ headersJSON }</pre>
			</details>
			<details open>
				<summary class="text-sm font-semibold cursor-pointer">Request Body</summary>
				<pre class="bg-base-200 rounded p-2 text-xs font-mono overflow-x-auto max-h-96 mt-1">{ bodyJSON }</pre>
			</details>
			if bodyTruncated {
				<div class="text-xs text-warning mt-1">Body truncated to 64KB</div>
			}
		</div>
	</td>
}
```

- [ ] **Step 3: Verify compilation**

Run: `templ generate pkg/views/partials/event_payload.templ pkg/views/partials/webhook_payload.templ`
Expected: no errors.

---

### Task 12: Create event_webservice.go route handlers

**Files:**
- Create: `internal/modules/web/event_webservice.go`

- [ ] **Step 1: Write `eventWebserviceRules` and handler functions**

```go
package web

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types"
	pages "github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
	"github.com/flowline-io/flowbot/pkg/webservice"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

var eventWebserviceRules = []webservice.Rule{
	webservice.Get("/events", eventsPage, route.WithNotAuth()),
	webservice.Get("/events/data-events", dataEventsTable, route.WithNotAuth()),
	webservice.Get("/events/webhook-logs", webhookLogsTable, route.WithNotAuth()),
	webservice.Get("/events/payload/:eventID", eventPayload, route.WithNotAuth()),
}

func requireAdmin(ctx fiber.Ctx) error {
	if !isAuthenticated(ctx) {
		return redirectToLogin(ctx)
	}
	scopes := route.GetScopes(ctx)
	if !auth.HasScope(scopes, auth.ScopeAdmin) {
		ctx.Status(fiber.StatusForbidden)
		return ctx.SendString("Admin access required")
	}
	return nil
}

// getPipelineDefinitionsForMatch loads published pipeline definitions from the store
// and parses them for FindByEvent matching.
func getPipelineDefinitionsForMatch(ctx context.Context) ([]pipeline.Definition, error) {
	s := getPipelineDefStore()
	if s == nil {
		return nil, nil
	}
	records, err := s.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	var defs []pipeline.Definition
	for _, rec := range records {
		if rec.YamlPublished == nil || *rec.YamlPublished == "" {
			continue
		}
		ed, err := pipeline.ParseEditorYAML(*rec.YamlPublished)
		if err != nil {
			continue
		}
		defs = append(defs, pipeline.ExpandDefinitions([]pipeline.EditorDefinition{*ed})...)
	}
	return defs, nil
}

func eventsPage(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	ctx.Type("html")
	return pages.EventsPage(pages.EventsPageParams{
		ActiveTab:  "data-events",
		Sources:    sources,
		EventTypes: eventTypes,
	}).Render(context.Background(), ctx.Response().BodyWriter())
}

func dataEventsTable(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sourceFilter := ctx.Query("source")
	typeFilter := ctx.Query("type")
	cursor := ctx.Query("cursor")

	s := store.NewEventStore(getStoreClient())
	events, nextCursor, err := s.ListDataEvents(context.Background(), store.ListDataEventsOptions{
		Limit:     20,
		Cursor:    cursor,
		Source:    sourceFilter,
		EventType: typeFilter,
	})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load events: " + err.Error()).Render(context.Background(), ctx.Response().BodyWriter())
	}

	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	ctx.Type("html")
	return partials.DataEventsTable(sources, eventTypes, sourceFilter, typeFilter, events, nextCursor).
		Render(context.Background(), ctx.Response().BodyWriter())
}

func webhookLogsTable(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sourceFilter := ctx.Query("source")
	typeFilter := ctx.Query("type")
	cursor := ctx.Query("cursor")

	s := store.NewEventStore(getStoreClient())
	events, nextCursor, err := s.ListDataEvents(context.Background(), store.ListDataEventsOptions{
		Limit:     20,
		Cursor:    cursor,
		Source:    sourceFilter,
		EventType: typeFilter,
		Webhook:   true,
	})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load webhook logs: " + err.Error()).Render(context.Background(), ctx.Response().BodyWriter())
	}

	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	ctx.Type("html")
	return partials.WebhookLogsTable(sources, eventTypes, sourceFilter, typeFilter, events, nextCursor).
		Render(context.Background(), ctx.Response().BodyWriter())
}

func eventPayload(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	eventID := ctx.Params("eventID")

	s := store.NewEventStore(getStoreClient())
	found, err := s.GetDataEventByEventID(context.Background(), eventID)
	if err != nil || found == nil {
		ctx.Type("html")
		return partials.EmptyState("Event not found").Render(context.Background(), ctx.Response().BodyWriter())
	}

	payloadJSON := "{}"
	if found.Data != nil {
		if b, err := sonic.Marshal(found.Data); err == nil {
			payloadJSON = string(b)
		}
	}

	// Check if this is a webhook event
	if found.Source == "webhook" || hasWebhookData(found) {
		headersJSON := "{}"
		bodyJSON := ""
		bodyTruncated := false
		if found.Data != nil {
			if h, ok := found.Data.(map[string]any)["_webhook_headers"]; ok {
				if b, err := sonic.Marshal(h); err == nil {
					headersJSON = string(b)
				}
			}
			if b, ok := found.Data.(map[string]any)["_webhook_body"]; ok {
				if s, ok := b.(string); ok {
					bodyJSON = s
				}
			}
			if t, ok := found.Data.(map[string]any)["_webhook_body_truncated"]; ok {
				if v, ok := t.(bool); ok {
					bodyTruncated = v
				}
			}
		}
		ctx.Type("html")
		return partials.WebhookPayloadDetail(headersJSON, bodyJSON, bodyTruncated).
			Render(context.Background(), ctx.Response().BodyWriter())
	}

	truncated := false
	ctx.Type("html")
	return partials.EventPayloadDetail(payloadJSON, truncated).
		Render(context.Background(), ctx.Response().BodyWriter())
}

func hasWebhookData(e *gen.DataEvent) bool {
	if e.Data == nil {
		return false
	}
	d, ok := e.Data.(map[string]any)
	if !ok {
		return false
	}
	_, hasMethod := d["_webhook_method"]
	return hasMethod
}

func getStoreClient() *store.Client {
	if store.Database == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return client
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/web/`
Expected: no errors (expect some due to missing templates — the tRPC generate step is next).

---

### Task 13: Create WebhookLogs table partial

**Files:**
- Create: `pkg/views/partials/webhook_logs_table.templ`

- [ ] **Step 1: Write `webhook_logs_table.templ`**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/gen"

func webhookSourceDisplay(e *gen.DataEvent) string {
	source := e.Source
	path := ""
	if e.Data != nil {
		if d, ok := e.Data.(map[string]any); ok {
			if p, ok := d["_webhook_path"]; ok {
				if s, ok := p.(string); ok {
					path = s
				}
			}
		}
	}
	if path != "" {
		return source + " " + path
	}
	return source
}

func webhookMethod(e *gen.DataEvent) string {
	if e.Data != nil {
		if d, ok := e.Data.(map[string]any); ok {
			if m, ok := d["_webhook_method"]; ok {
				if s, ok := m.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

templ WebhookLogsTable(sources, eventTypes []string, sourceFilter, typeFilter string, events []*gen.DataEvent, nextCursor string) {
	<div class="card bg-base-100 shadow-sm" id="webhook-logs-table" data-testid="webhook-logs-table">
		<form class="flex flex-wrap items-end gap-2 p-3 bg-base-200 rounded-t-box"
			hx-get="/service/web/events/webhook-logs"
			hx-target="#webhook-logs-table"
			hx-swap="outerHTML"
			hx-push-url="true"
			hx-trigger="submit">
			<label class="form-control max-w-xs">
				<div class="label"><span class="label-text text-xs">Source</span></div>
				<select name="source" class="select select-bordered select-sm">
					<option value="">All</option>
					for _, s := range sources {
						<option value={ s } selected={ templ.KV(s == sourceFilter, true) }>{ s }</option>
					}
				</select>
			</label>
			<label class="form-control max-w-xs">
				<div class="label"><span class="label-text text-xs">Event Type</span></div>
				<select name="type" class="select select-bordered select-sm">
					<option value="">All</option>
					for _, t := range eventTypes {
						<option value={ t } selected={ templ.KV(t == typeFilter, true) }>{ t }</option>
					}
				</select>
			</label>
			<button type="submit" class="btn btn-sm btn-primary" data-testid="filter-apply">Apply</button>
		</form>

		<div class="overflow-x-auto">
			<table class="table">
				<thead>
				<tr>
					<th class="text-xs uppercase">Time</th>
					<th class="text-xs uppercase">Source</th>
					<th class="text-xs uppercase">Path</th>
					<th class="text-xs uppercase">Method</th>
					<th class="text-xs uppercase">Status</th>
					<th class="text-xs uppercase">Pipeline</th>
				</tr>
				</thead>
				<tbody>
				for _, e := range events {
					<tbody id={ "wh-" + e.EventID }>
						<tr class="cursor-pointer hover"
							hx-get={ templ.URL("/service/web/events/payload/" + e.EventID) }
							hx-trigger="click"
							hx-target={ "#wh-detail-" + e.EventID }
							hx-swap="innerHTML show:top">
							<td class="text-xs whitespace-nowrap">{ e.CreatedAt.Format("15:04:05") }</td>
							<td class="text-xs">{ webhookSourceDisplay(e) }</td>
							<td class="text-xs font-mono">{ webhookPath(e) }</td>
							<td><span class="badge badge-sm">{ webhookMethod(e) }</span></td>
							<td><span class="badge badge-success badge-sm">202</span></td>
							<td class="text-xs"><span class="text-base-content/50">loading...</span></td>
						</tr>
						<tr id={ "wh-detail-" + e.EventID } class="hidden">
							<td colspan="6"></td>
						</tr>
					</tbody>
				}
				if len(events) == 0 {
					<tr id="wh-events-empty">
						<td colspan="6" class="text-center text-base-content/50 py-4">No webhook receipts recorded yet.</td>
					</tr>
				}
				</tbody>
			</table>
		</div>

		if nextCursor != "" {
			<div class="p-3 text-center" id="load-more-container">
				<button class="btn btn-sm btn-ghost"
					hx-get={ templ.URL("/service/web/events/webhook-logs?source=" + sourceFilter + "&type=" + typeFilter + "&cursor=" + nextCursor) }
					hx-target="#load-more-container"
					hx-swap="outerHTML"
					data-testid="load-more">Load more</button>
			</div>
		}
	</div>
}
```

- [ ] **Step 2: Add `webhookPath` helper**

```go
func webhookPath(e *gen.DataEvent) string {
	if e.Data != nil {
		if d, ok := e.Data.(map[string]any); ok {
			if p, ok := d["_webhook_path"]; ok {
				if s, ok := p.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}
```

- [ ] **Step 3: Verify compilation**

Run: `templ generate pkg/views/partials/webhook_logs_table.templ`
Expected: no errors.

---

### Task 14: Wire event routes into module and add nav link

**Files:**
- Modify: `internal/modules/web/module.go`
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Wire event routes in module.go**

In `module.go`, add `eventWebserviceRules` to the Webservice mounting and Rules():

```go
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	module.Webservice(app, Name, webserviceRules)
	module.Webservice(app, Name, pipelineWebserviceRules)
	module.Webservice(app, Name, viewWebserviceRules)
	module.Webservice(app, Name, eventWebserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{commandRules, formRules, webserviceRules, pipelineWebserviceRules, viewWebserviceRules, eventWebserviceRules}
}
```

- [ ] **Step 2: Add nav link in base.templ**

In `pkg/views/layout/base.templ`, add to the navbar-end div:

```html
<a href="/service/web/events" data-testid="nav-events" class="btn btn-ghost btn-sm">Events</a>
```

Place it between Pipelines and Configs links.

- [ ] **Step 3: Verify compilation**

Run: `templ generate pkg/views/layout/base.templ && go build ./internal/modules/web/`
Expected: no errors.

---

### Task 15: Write route handler unit tests

**Files:**
- Create: `internal/modules/web/event_webservice_test.go`

- [ ] **Step 1: Write unit tests for helper functions**

The `requireAdmin` and event-page handlers depend on the full app context (store, templates). Test the pure helper functions instead:

```go
package web

import (
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestHasWebhookData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		event *gen.DataEvent
		want  bool
	}{
		{
			name:  "nil data returns false",
			event: &gen.DataEvent{},
			want:  false,
		},
		{
			name: "has _webhook_method returns true",
			event: &gen.DataEvent{
				Data: schema.JSON(map[string]any{"_webhook_method": "POST"}),
			},
			want: true,
		},
		{
			name: "no webhook keys returns false",
			event: &gen.DataEvent{
				Data: schema.JSON(map[string]any{"foo": "bar"}),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasWebhookData(tt.event))
		})
	}
}

func TestGetStoreClient_Nil(t *testing.T) {
	t.Parallel()
	// store.Database is nil in unit tests (no DB initialized)
	client := getStoreClient()
	assert.Nil(t, client)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/modules/web/ -run TestHasWebhookData -v -count=1`
Expected: PASS.

---

### Task 16: Hydrate filter cache on module startup

**Files:**
- Modify: `internal/modules/web/module.go`

- [ ] **Step 1: In `Init()` or `Bootstrap()`, hydrate the filter cache**

In the `moduleHandler.Bootstrap()` method, add:

```go
func (h *moduleHandler) Bootstrap() error {
	if !h.initialized {
		return nil
	}
	// Hydrate event filter cache from database
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			es := store.NewEventStore(client)
			sources, err := es.ListDistinctEventSources(context.Background(), 30*24*time.Hour)
			if err == nil {
				types, err2 := es.ListDistinctEventTypes(context.Background(), 30*24*time.Hour)
				if err2 == nil {
					types.EventFilterCache.Hydrate(sources, types)
				}
			}
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Add imports: `"context"`, `"time"`, `"github.com/flowline-io/flowbot/pkg/types"`, `"github.com/flowline-io/flowbot/internal/store"`

Run: `go build ./internal/modules/web/`
Expected: no errors.

---

### Task 17: Add `EmptyState` helper partial

**Files:**
- Create: `pkg/views/partials/empty_state.templ`

- [ ] **Step 1: Write `empty_state.templ`**

```templ
package partials

templ EmptyState(message string) {
	<div class="card bg-base-100 shadow-sm">
		<div class="p-6 text-center text-base-content/50">
			{ message }
		</div>
	</div>
}
```

- [ ] **Step 2: Generate**

Run: `templ generate pkg/views/partials/empty_state.templ`
Expected: no errors.

---

### Task 18: Generate all templ files

**Files:**
- All `.templ` files under `pkg/views/`

- [ ] **Step 1: Run templ generate**

Run: `templ generate pkg/views/...`
Expected: no errors.

- [ ] **Step 2: Verify full build**

Run: `go build ./...`
Expected: no errors.

---

### Task 19: Run linter and fix issues

**Files:**
- All modified/created files

- [ ] **Step 1: Run linter**

Run: `go tool task lint`
Expected: resolve any warnings.

- [ ] **Step 2: Run unit tests**

Run: `go tool task test`
Expected: all tests pass.

---

### Task 20: Add DB indexes via ent migration hook

**Files:**
- Modify: `internal/store/store.go` (or ent schema annotation)

- [ ] **Step 1: Add index annotations to `data_event.go` schema**

In `internal/store/ent/schema/data_event.go`, update the `Indexes()` method:

```go
func (DataEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type"),
		index.Fields("tags").Annotations(entsql.IndexType("GIN")),
		// Cursor-based pagination
		index.Fields("created_at").Annotations(entsql.Desc()),
		// Filtered pagination by source
		index.Fields("source", "created_at").Annotations(entsql.Desc()),
	}
}
```

- [ ] **Step 2: Re-generate ent code**

Run: `go tool task ent`
Expected: ent schemas regenerated.

- [ ] **Step 3: Create raw SQL migration for partial index**

The partial webhook index cannot be expressed in ent annotations. Add a migration step. Create `internal/store/postgres/migration.go` or add to an existing migration file:

```sql
CREATE INDEX IF NOT EXISTS idx_data_events_webhook ON data_events (created_at DESC, id DESC)
WHERE data->>'_webhook_method' IS NOT NULL;
```

This is run manually or via the postgres adapter's startup hook.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: no errors.

---

### Task 21: BDD acceptance tests (placeholder)

**Files:**
- Create: `tests/specs/events_page_spec_test.go`

- [ ] **Step 1: Write Ginkgo spec skeleton**

BDD tests require the full application stack (PostgreSQL, Redis, modules). This task defines the spec structure; full implementation follows after unit/integration tests pass.

```go
package specs_test

import (
	"github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.Describe("Events Page", func() {
	ginkgo.Context("Data Events tab", func() {
		ginkgo.It("renders the events page with Data Events tab active")
		ginkgo.It("shows data_events records in the table")
		ginkgo.It("filters by source dropdown")
		ginkgo.It("loads more rows with cursor pagination")
	})

	ginkgo.Context("Webhook Logs tab", func() {
		ginkgo.It("switches to Webhook Logs tab")
		ginkgo.It("shows webhook events filtered by _webhook_method")
	})

	ginkgo.Context("Admin access control", func() {
		ginkgo.It("redirects unauthenticated users to login")
		ginkgo.It("returns 403 for non-admin users")
		ginkgo.It("allows admin users to view the page")
	})
})
```

- [ ] **Step 2: Verify spec compiles**

Run: `go build ./tests/specs/`
Expected: no errors.

---

### Task 22: Final verification

- [ ] **Step 1: Run full test suite**

```bash
go tool task test
go tool task lint
go tool task build
```

- [ ] **Step 2: Verify all commands pass**

All three commands should exit with 0.
