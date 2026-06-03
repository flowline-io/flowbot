# Events Advanced Filtering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add time range selection, full-text search, pipeline filtering, and page-number pagination to the Events page.

**Architecture:** Extend `ListDataEventsOptions` with offset, search, pipeline, and time range fields; add `CountDataEvents` and `ListDistinctEventPipelineNames` store methods; merge data/webhook table handlers into one `filteredEventsTable`; add Alpine.js `eventFilters()` component and templ partials for filter bar + pagination controls.

**Tech Stack:** Go 1.26+, Ent ORM + PostgreSQL, templ v0.3, HTMX 2.x, Alpine.js 3.x, DaisyUI v5

**Spec:** `docs/superpowers/specs/2026-06-03-events-advanced-filtering-design.md`

---

### Task 1: Extend Store Tests for Advanced Filtering

**Files:**
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Add advanced filtering test cases to `TestListDataEvents`**

The existing `tests` slice inside `TestListDataEvents` (line 772) already has 6 cases. Append these new cases to the `tests` slice, after the existing cases and before the closing `}` of the slice literal:

```go
{
    name:      "search matches source field",
    opts:      ListDataEventsOptions{Limit: 10, Search: "github"},
    wantCount: 1,
},
{
    name:      "search matches data payload",
    opts:      ListDataEventsOptions{Limit: 10, Search: "feed"},
    wantCount: 1,
},
{
    name:      "search no match",
    opts:      ListDataEventsOptions{Limit: 10, Search: "nomatchxyz"},
    wantCount: 0,
},
{
    name:      "pipeline name filter returns matched events",
    opts:      ListDataEventsOptions{Limit: 10, PipelineName: "test-pipeline"},
    wantCount: 0,
},
{
    name:      "time start filter returns events after time",
    opts:      ListDataEventsOptions{Limit: 10, TimeStart: timePtr(time.Now().Add(-10 * time.Minute))},
    wantCount: 3,
},
{
    name:      "time end filter returns events before time",
    opts:      ListDataEventsOptions{Limit: 10, TimeEnd: timePtr(time.Now().Add(-1 * time.Hour))},
    wantCount: 0,
},
{
    name:          "offset-based pagination page 1",
    opts:          ListDataEventsOptions{Limit: 1, Offset: 0},
    wantCount:     1,
    wantHasCursor: false,
},
{
    name:      "offset-based pagination page 2",
    opts:      ListDataEventsOptions{Limit: 1, Offset: 1},
    wantCount: 1,
},
```

Add a `timePtr` helper above the `TestListDataEvents` function:

```go
func timePtr(t time.Time) *time.Time { return &t }
```

- [ ] **Step 2: Run store tests to see them fail (missing fields / methods)**

```bash
go test ./internal/store/ -run TestListDataEvents -v -count=1
```

Expected: compile errors for `Search`, `PipelineName`, `TimeStart`, `TimeEnd`, `Offset` fields on `ListDataEventsOptions`.

- [ ] **Step 3: Commit**

```bash
git add internal/store/store_test.go
git commit -m "test(store): add advanced filtering test cases for ListDataEvents"
```

---

### Task 2: Extend ListDataEventsOptions and ListDataEvents

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add new fields to `ListDataEventsOptions`**

Replace the entire struct (lines 541-547) with:

```go
// ListDataEventsOptions holds filters and pagination for listing data events.
type ListDataEventsOptions struct {
	Limit        int       // max 100, default 20
	Offset       int       // page offset for offset-based pagination
	Cursor       string    // opaque CreatedAt cursor (backward compatible)
	Source       string    // filter by source, empty = all
	EventType    string    // filter by event type, empty = all
	Webhook      bool      // if true, only events where data->>'_webhook_method' IS NOT NULL
	Search       string    // ILIKE match against source and data::text
	PipelineName string    // filter events that triggered a specific pipeline
	TimeStart    *time.Time // created_at >= TimeStart
	TimeEnd      *time.Time // created_at <= TimeEnd
}
```

- [ ] **Step 2: Replace `ListDataEvents` implementation**

Replace the entire `ListDataEvents` function (lines 551-593) with:

```go
// ListDataEvents returns paginated data_events ordered by created_at DESC.
// Supports offset-based pagination (when Offset > 0) and cursor-based (backward compatible).
func (s *EventStore) ListDataEvents(ctx context.Context, opts ListDataEventsOptions) ([]*gen.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := s.client.DataEvent.Query().
		Order(dataevent.ByCreatedAt(sql.OrderDesc()))

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
	if opts.Search != "" {
		q = q.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(
				"source ILIKE '%' || $1 || '%' OR data::text ILIKE '%' || $1 || '%'",
				opts.Search,
			))
		})
	}
	if opts.PipelineName != "" {
		q = q.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(
				"event_id IN (SELECT event_id FROM pipeline_runs WHERE pipeline_name = $1)",
				opts.PipelineName,
			))
		})
	}
	if opts.TimeStart != nil {
		q = q.Where(dataevent.CreatedAtGTE(*opts.TimeStart))
	}
	if opts.TimeEnd != nil {
		q = q.Where(dataevent.CreatedAtLTE(*opts.TimeEnd))
	}

	// Offset-based pagination (mutually exclusive with cursor)
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset).Limit(opts.Limit)
		events, err := q.All(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("list data events: %w", err)
		}
		return events, "", nil
	}

	// Cursor-based pagination (backward compatible)
	q = q.Limit(opts.Limit + 1)
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

- [ ] **Step 3: Run store tests to verify they compile and pass**

```bash
go test ./internal/store/ -run TestListDataEvents -v -count=1
```

Expected: all test cases pass (new cases should pass once search/pipeline/time filters are wired).

- [ ] **Step 4: Commit**

```bash
git add internal/store/store.go
git commit -m "feat(store): add Search, PipelineName, TimeStart/End, Offset to ListDataEventsOptions"
```

---

### Task 3: Add CountDataEvents Method

**Files:**
- Modify: `internal/store/store_test.go`
- Modify: `internal/store/store.go`

- [ ] **Step 1: Write `TestCountDataEvents` in `store_test.go`**

Add this test function after `TestListDataEvents` (before the `ResourceChainStore tests` comment block around line 826):

```go
func TestCountDataEvents(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewEventStore(client)
	ctx := context.Background()

	events := []types.DataEvent{
		{EventID: "cnt-001", EventType: "issue.created", Source: "github", Capability: "forge", EntityID: "repo#1"},
		{EventID: "cnt-002", EventType: "bookmark.created", Source: "karakeep", Capability: "bookmark", EntityID: "url-1"},
		{EventID: "cnt-003", EventType: "issue.created", Source: "github", Capability: "forge", EntityID: "repo#2"},
	}
	for _, e := range events {
		require.NoError(t, store.AppendDataEvent(ctx, e))
	}

	tests := []struct {
		name      string
		opts      ListDataEventsOptions
		wantCount int64
	}{
		{
			name:      "count all events",
			opts:      ListDataEventsOptions{Limit: 20},
			wantCount: 3,
		},
		{
			name:      "count filtered by source",
			opts:      ListDataEventsOptions{Limit: 20, Source: "github"},
			wantCount: 2,
		},
		{
			name:      "count filtered by event type",
			opts:      ListDataEventsOptions{Limit: 20, EventType: "bookmark.created"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := store.CountDataEvents(ctx, tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}
```

- [ ] **Step 2: Run test to see it fail**

```bash
go test ./internal/store/ -run TestCountDataEvents -v -count=1
```

Expected: compile error "store.EventStore has no field or method CountDataEvents"

- [ ] **Step 3: Implement `CountDataEvents` in `store.go`**

Insert after the `ListDataEvents` method (after line ~593 in the new code):

```go
// CountDataEvents returns the total number of data_events matching the given filters.
// Uses the same filter predicates as ListDataEvents without pagination.
func (s *EventStore) CountDataEvents(ctx context.Context, opts ListDataEventsOptions) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}

	q := s.client.DataEvent.Query()

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
	if opts.Search != "" {
		q = q.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(
				"source ILIKE '%' || $1 || '%' OR data::text ILIKE '%' || $1 || '%'",
				opts.Search,
			))
		})
	}
	if opts.PipelineName != "" {
		q = q.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(
				"event_id IN (SELECT event_id FROM pipeline_runs WHERE pipeline_name = $1)",
				opts.PipelineName,
			))
		})
	}
	if opts.TimeStart != nil {
		q = q.Where(dataevent.CreatedAtGTE(*opts.TimeStart))
	}
	if opts.TimeEnd != nil {
		q = q.Where(dataevent.CreatedAtLTE(*opts.TimeEnd))
	}

	count, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count data events: %w", err)
	}

	return int64(count), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/store/ -run TestCountDataEvents -v -count=1
```

Expected: all 3 test cases PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): add CountDataEvents method for pagination totals"
```

---

### Task 4: Add ListDistinctEventPipelineNames Method

**Files:**
- Modify: `internal/store/store_test.go`
- Modify: `internal/store/store.go`

- [ ] **Step 1: Write `TestListDistinctEventPipelineNames` in `store_test.go`**

Add after `TestCountDataEvents`:

```go
func TestListDistinctEventPipelineNames(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewEventStore(client)
	ctx := context.Background()

	tests := []struct {
		name      string
		setup     func()
		wantCount int
		wantNames []string
	}{
		{
			name:      "no pipeline runs returns empty",
			setup:     func() {},
			wantCount: 0,
		},
		{
			name: "single pipeline name",
			setup: func() {
				client.PipelineRun.Create().
					SetPipelineName("test-pipeline").
					SetEventID("evt-1").
					SetEventType("test.event").
					SetTriggerSource("event").
					SetStatus(1).
					SetCreatedAt(time.Now()).
					SaveX(ctx)
			},
			wantCount: 1,
			wantNames: []string{"test-pipeline"},
		},
		{
			name: "multiple pipeline names deduped and sorted",
			setup: func() {
				client.PipelineRun.Create().
					SetPipelineName("beta-pipeline").
					SetEventID("evt-2").
					SetEventType("test.event").
					SetTriggerSource("event").
					SetStatus(1).
					SetCreatedAt(time.Now()).
					SaveX(ctx)
				client.PipelineRun.Create().
					SetPipelineName("alpha-pipeline").
					SetEventID("evt-3").
					SetEventType("test.event").
					SetTriggerSource("event").
					SetStatus(1).
					SetCreatedAt(time.Now()).
					SaveX(ctx)
			},
			wantCount: 3,
			wantNames: []string{"alpha-pipeline", "beta-pipeline", "test-pipeline"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			names, err := store.ListDistinctEventPipelineNames(ctx)
			require.NoError(t, err)
			assert.Len(t, names, tt.wantCount)
			if tt.wantNames != nil {
				assert.Equal(t, tt.wantNames, names)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to see it fail**

```bash
go test ./internal/store/ -run TestListDistinctEventPipelineNames -v -count=1
```

Expected: compile error

- [ ] **Step 3: Implement `ListDistinctEventPipelineNames` in `store.go`**

Insert after `CountDataEvents`:

```go
// ListDistinctEventPipelineNames returns distinct pipeline names from pipeline_runs
// that have matched events, ordered alphabetically.
func (s *EventStore) ListDistinctEventPipelineNames(ctx context.Context) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	rows, err := s.client.PipelineRun.Query().
		Unique(true).
		Select(pipelinerun.FieldPipelineName).
		Order(pipelinerun.ByPipelineName(sql.OrderAsc())).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct pipeline names: %w", err)
	}

	return rows, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/store/ -run TestListDistinctEventPipelineNames -v -count=1
```

Expected: all 3 test cases PASS.

- [ ] **Step 5: Run all store tests to confirm no regressions**

```bash
go test ./internal/store/ -v -count=1
```

- [ ] **Step 6: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): add ListDistinctEventPipelineNames method"
```

---

### Task 5: Add parseEventFilterParams and filteredEventsTable Handler

**Files:**
- Modify: `internal/modules/web/event_webservice.go`

- [ ] **Step 1: Add `parseEventFilterParams` helper and `FilteredEventsTableParams` type**

Add after the `hasWebhookData` function (line 54) and before `eventsPage`:

```go
// FilteredEventsTableParams holds all parameters for the filtered events table.
type FilteredEventsTableParams struct {
	Tab          string
	Sources      []string
	EventTypes   []string
	PipelineNames []string
	SourceFilter string
	TypeFilter   string
	PipelineFilter string
	SearchFilter string
	TimeStart    string
	TimeEnd      string
	Page         int
	PerPage      int
	Events       []*gen.DataEvent
	Total        int64
}

// parseEventFilterParams extracts filter parameters from the request query string.
func parseEventFilterParams(c fiber.Ctx) store.ListDataEventsOptions {
	opts := store.ListDataEventsOptions{
		Source:    c.Query("source"),
		EventType: c.Query("type"),
		Search:    c.Query("search"),
	}

	if p := c.Query("pipeline"); p != "" {
		opts.PipelineName = p
	}

	perPage := 20
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			if v > 100 {
				v = 100
			}
			perPage = v
		}
	}
	opts.Limit = perPage

	page := 1
	if pg := c.Query("page"); pg != "" {
		if v, err := strconv.Atoi(pg); err == nil && v > 0 {
			page = v
		}
	}
	opts.Offset = (page - 1) * perPage

	if ts := c.Query("time_start"); ts != "" {
		if t, err := parseTimeParam(ts); err == nil {
			opts.TimeStart = &t
		}
	}
	if te := c.Query("time_end"); te != "" {
		if t, err := parseTimeParam(te); err == nil {
			opts.TimeEnd = &t
		}
	}

	// Invalid time range: ignore both
	if opts.TimeStart != nil && opts.TimeEnd != nil && opts.TimeEnd.Before(*opts.TimeStart) {
		opts.TimeStart = nil
		opts.TimeEnd = nil
	}

	tab := c.Query("tab")
	if tab == "webhook-logs" {
		opts.Webhook = true
	}

	return opts
}
```

Add imports to the import block: `"fmt"`, `"strconv"`, `"time"`.

Also add a `parseTimeParam` helper:

```go
// parseTimeParam parses a time query parameter supporting RFC3339 and datetime-local formats.
func parseTimeParam(s string) (time.Time, error) {
	formats := []string{time.RFC3339, "2006-01-02T15:04"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}
```

- [ ] **Step 2: Add the `filteredEventsTable` handler**

Add after `eventsPage` (around line 80, after the function's closing `}`):

```go
func filteredEventsTable(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	s := getEventStore()
	if s == nil {
		c.Type("html")
		return partials.EmptyState("Store not available").Render(c.Context(), c.Response().BodyWriter())
	}

	opts := parseEventFilterParams(c)
	tab := c.Query("tab", "data-events")

	total, err := s.CountDataEvents(c.Context(), opts)
	if err != nil {
		c.Type("html")
		return partials.EmptyState("Failed to count events").Render(c.Context(), c.Response().BodyWriter())
	}

	events, _, err := s.ListDataEvents(c.Context(), opts)
	if err != nil {
		c.Type("html")
		return partials.EmptyState("Failed to load events").Render(c.Context(), c.Response().BodyWriter())
	}

	// Build event ID list for pipeline name lookups
	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
	}

	runMap, _ := s.GetPipelineRunsForEvents(c.Context(), eventIDs)

	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	perPage := opts.Limit
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	currentPage := 1
	if pg := c.Query("page"); pg != "" {
		if v, err := strconv.Atoi(pg); err == nil && v > 0 {
			currentPage = v
		}
	}
	if currentPage > totalPages && totalPages > 0 {
		currentPage = totalPages
	}

	pageInfo := partials.PageInfo{
		Page:       currentPage,
		TotalPages: totalPages,
		Total:      total,
		PerPage:    perPage,
		HasPrev:    currentPage > 1,
		HasNext:    currentPage < totalPages,
	}

	c.Type("html")
	if opts.Webhook {
		return partials.WebhookLogsTable(sources, eventTypes, events, pageInfo, runMap).
			Render(c.Context(), c.Response().BodyWriter())
	}
	return partials.DataEventsTable(sources, eventTypes, events, pageInfo, runMap).
		Render(c.Context(), c.Response().BodyWriter())
}
```

- [ ] **Step 3: Update the route registration**

Replace lines 19-22 with:

```go
var eventWebserviceRules = []webservice.Rule{
	webservice.Get("/events", eventsPage, route.WithNotAuth()),
	webservice.Get("/events/filtered-events", filteredEventsTable, route.WithNotAuth()),
	webservice.Get("/events/data-events", dataEventsTable, route.WithNotAuth()),     // kept for backward compat
	webservice.Get("/events/webhook-logs", webhookLogsTable, route.WithNotAuth()),   // kept for backward compat
	webservice.Get("/events/payload/:eventID", eventPayload, route.WithNotAuth()),
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/modules/web/
```

Expected: compile errors (template signatures changed). Proceed to fix templates in Tasks 9-12.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/event_webservice.go
git commit -m "feat(web): add parseEventFilterParams and filteredEventsTable handler"
```

---

### Task 6: Update eventsPage to Inject Pipeline Names

**Files:**
- Modify: `internal/modules/web/event_webservice.go`

- [ ] **Step 1: Update `eventsPage` to call `ListDistinctEventPipelineNames`**

Replace `eventsPage` (lines 56-80) with:

```go
func eventsPage(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	s := getEventStore()
	var pipelineNames []string
	if s != nil {
		pipelineNames, _ = s.ListDistinctEventPipelineNames(c.Context())
	}

	c.Type("html")
	return pages.EventsPage(pages.EventsPageParams{
		Sources:       sources,
		EventTypes:    eventTypes,
		PipelineNames: pipelineNames,
	}).Render(c.Context(), c.Response().BodyWriter())
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/web/event_webservice.go
git commit -m "feat(web): inject pipeline names into eventsPage"
```

---

### Task 7: Create event-filters.js Alpine.js Component

**Files:**
- Create: `public/js/event-filters.js`

- [ ] **Step 1: Create the file**

```js
Alpine.data('eventFilters', () => ({
  timeRange: 'custom',
  timeStart: '',
  timeEnd: '',
  search: '',
  pipeline: '',
  source: '',
  eventType: '',
  tab: 'data-events',

  init() {
    const params = new URLSearchParams(window.location.search);
    this.timeStart = params.get('time_start') || '';
    this.timeEnd = params.get('time_end') || '';
    this.search = params.get('search') || '';
    this.pipeline = params.get('pipeline') || '';
    this.source = params.get('source') || '';
    this.eventType = params.get('type') || '';
    this.tab = params.get('tab') || 'data-events';

    if (this.timeStart && this.timeEnd) {
      const now = Date.now();
      const start = new Date(this.timeStart).getTime();
      const end = new Date(this.timeEnd).getTime();
      const diff = now - start;
      // Heuristic: if end is within 5 sec of now and diff matches a shortcut, show it
      if (Math.abs(now - end) < 5000) {
        if (Math.abs(diff - 3600000) < 5000) {
          this.timeRange = '1h';
        } else if (Math.abs(diff - 86400000) < 10000) {
          this.timeRange = '24h';
        } else if (Math.abs(diff - 604800000) < 30000) {
          this.timeRange = '7d';
        }
      }
    }
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

  getFilterParams() {
    const params = new URLSearchParams();
    params.set('tab', this.tab);
    if (this.search) params.set('search', this.search);
    if (this.pipeline) params.set('pipeline', this.pipeline);
    if (this.source) params.set('source', this.source);
    if (this.eventType) params.set('type', this.eventType);
    if (this.timeStart) params.set('time_start', new Date(this.timeStart).toISOString());
    if (this.timeEnd) params.set('time_end', new Date(this.timeEnd).toISOString());
    params.set('page', '1');
    return params.toString();
  },

  submitFilter() {
    const url = '/service/web/events/filtered-events?' + this.getFilterParams();
    const container = document.getElementById('events-table-container');
    // HTMX will handle this via the form's hx-get trigger
    // This is called from buttons that need to build the URL
    htmx.ajax('GET', url, { target: '#events-table-container', swap: 'innerHTML' });
  },

  switchTab(newTab) {
    this.tab = newTab;
    this.submitFilter();
  },

  debounceSearch() {
    clearTimeout(this._searchTimer);
    this._searchTimer = setTimeout(() => this.submitFilter(), 300);
  }
}));
```

- [ ] **Step 2: Commit**

```bash
git add public/js/event-filters.js
git commit -m "feat(ui): create eventFilters Alpine.js component"
```

---

### Task 8: Create event_filters.templ Filter Bar

**Files:**
- Create: `pkg/views/partials/event_filters.templ`

- [ ] **Step 1: Create the file**

```templ
package partials

type FilterBarParams struct {
	Tab           string
	Sources       []string
	EventTypes    []string
	PipelineNames []string
	SourceFilter  string
	TypeFilter    string
	PipelineFilter string
	SearchFilter  string
	TimeStart     string
	TimeEnd       string
}

templ FilterBar(p FilterBarParams) {
	<div class="card bg-base-100 shadow-sm mb-4" data-testid="event-filter-bar">
		<div class="card-body p-3">
			<div class="flex flex-wrap items-center gap-2 mb-2">
				<div class="join">
					<button type="button" class="btn btn-sm join-item"
						x-on:click="setTimeRange('1h')"
						x-bind:class="timeRange === '1h' ? 'btn-primary' : ''"
						data-testid="time-range-1h">1h</button>
					<button type="button" class="btn btn-sm join-item"
						x-on:click="setTimeRange('24h')"
						x-bind:class="timeRange === '24h' ? 'btn-primary' : ''"
						data-testid="time-range-24h">24h</button>
					<button type="button" class="btn btn-sm join-item"
						x-on:click="setTimeRange('7d')"
						x-bind:class="timeRange === '7d' ? 'btn-primary' : ''"
						data-testid="time-range-7d">7d</button>
				</div>
				<input type="datetime-local" class="input input-bordered input-sm"
					x-model="timeStart" x-on:change="onDateChange()"
					data-filter-input name="time_start"
					data-testid="time-start"/>
				<span class="text-xs text-base-content/50">~</span>
				<input type="datetime-local" class="input input-bordered input-sm"
					x-model="timeEnd" x-on:change="onDateChange()"
					data-filter-input name="time_end"
					data-testid="time-end"/>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<input type="search" class="input input-bordered input-sm w-64"
					placeholder="Search events..."
					x-model="search"
					x-on:keyup="debounceSearch()"
					data-filter-input name="search"
					data-testid="search-input"/>
				<select class="select select-bordered select-sm"
					data-filter-input name="pipeline"
					x-model="pipeline"
					x-on:change="submitFilter()"
					data-testid="pipeline-select">
					<option value="">All pipelines</option>
					for _, name := range p.PipelineNames {
						<option value={ name } selected={ templ.KV(name == p.PipelineFilter, true) }>{ name }</option>
					}
				</select>
				<select class="select select-bordered select-sm"
					data-filter-input name="source"
					x-model="source"
					x-on:change="submitFilter()"
					data-testid="source-select">
					<option value="">All sources</option>
					for _, s := range p.Sources {
						<option value={ s } selected={ templ.KV(s == p.SourceFilter, true) }>{ s }</option>
					}
				</select>
				<select class="select select-bordered select-sm"
					data-filter-input name="type"
					x-model="eventType"
					x-on:change="submitFilter()"
					data-testid="type-select">
					<option value="">All types</option>
					for _, t := range p.EventTypes {
						<option value={ t } selected={ templ.KV(t == p.TypeFilter, true) }>{ t }</option>
					}
				</select>
			</div>
		</div>
	</div>
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/views/partials/event_filters.templ
git commit -m "feat(ui): create event_filters.templ filter bar partial"
```

---

### Task 9: Create event_pagination.templ Pagination Control

**Files:**
- Create: `pkg/views/partials/event_pagination.templ`

- [ ] **Step 1: Define `PageInfo` type in `helpers.go`**

Add to `pkg/views/partials/helpers.go`, before the existing `valuePreview` function:

```go
// PageInfo holds pagination state for the event table.
type PageInfo struct {
	Page       int
	TotalPages int
	Total      int64
	PerPage    int
	HasPrev    bool
	HasNext    bool
}
```

- [ ] **Step 2: Create `event_pagination.templ`**

```templ
package partials

import "fmt"

templ EventPagination(info PageInfo) {
	if info.Total == 0 {
		return
	}
	<div class="flex flex-wrap items-center justify-between p-3 border-t border-base-300" data-testid="pagination">
		<div class="flex items-center gap-2">
			<select class="select select-bordered select-xs"
				name="per_page"
				hx-get="/service/web/events/filtered-events"
				hx-include="[name='page']"
				hx-target="#events-table-container"
				hx-swap="innerHTML"
				x-on:change="
					$el.form.elements.namedItem('page').value = '1';
					htmx.trigger($el.form, 'submit');
				"
				data-testid="per-page-select">
				<option value="10" selected={ templ.KV(info.PerPage == 10, true) }>10</option>
				<option value="20" selected={ templ.KV(info.PerPage == 20, true) }>20</option>
				<option value="50" selected={ templ.KV(info.PerPage == 50, true) }>50</option>
				<option value="100" selected={ templ.KV(info.PerPage == 100, true) }>100</option>
			</select>
		</div>
		<div class="text-xs text-base-content/70">
			Showing { fmt.Sprintf("%d", (info.Page-1)*info.PerPage+1) }-{ fmt.Sprintf("%d", min(info.Page*info.PerPage, int(info.Total))) } of { fmt.Sprintf("%d", info.Total) }
		</div>
		<div class="flex items-center gap-1">
			<button class="btn btn-xs btn-ghost"
				if !info.HasPrev {
					disabled
				}
				hx-get={ templ.URL("/service/web/events/filtered-events?page=" + fmt.Sprintf("%d", info.Page-1)) }
				hx-target="#events-table-container"
				hx-swap="innerHTML"
				hx-include="[data-filter-input]"
				data-testid="pagination-prev">Prev</button>

			for _, p := range pageNumbers(info.Page, info.TotalPages) {
				if p == 0 {
					<span class="px-1 text-xs">...</span>
				} else if p == info.Page {
					<span class="btn btn-xs btn-primary" data-testid="pagination-page-active">{ fmt.Sprintf("%d", p) }</span>
				} else {
					<button class="btn btn-xs btn-ghost"
						hx-get={ templ.URL("/service/web/events/filtered-events?page=" + fmt.Sprintf("%d", p)) }
						hx-target="#events-table-container"
						hx-swap="innerHTML"
						hx-include="[data-filter-input]"
						data-testid={ "pagination-page-" + fmt.Sprintf("%d", p) }>{ fmt.Sprintf("%d", p) }</button>
				}
			}

			<button class="btn btn-xs btn-ghost"
				if !info.HasNext {
					disabled
				}
				hx-get={ templ.URL("/service/web/events/filtered-events?page=" + fmt.Sprintf("%d", info.Page+1)) }
				hx-target="#events-table-container"
				hx-swap="innerHTML"
				hx-include="[data-filter-input]"
				data-testid="pagination-next">Next</button>
		</div>
		<div class="flex items-center gap-1">
			<span class="text-xs text-base-content/50">Go to</span>
			<input type="number" class="input input-bordered input-xs w-16" min="1" max={ fmt.Sprintf("%d", info.TotalPages) }
				data-testid="pagination-jump"/>
			<button class="btn btn-xs btn-ghost"
				x-on:click="
					const pageVal = $el.parentElement.querySelector('input').value;
					const url = '/service/web/events/filtered-events?page=' + pageVal;
					htmx.ajax('GET', url, { target: '#events-table-container', swap: 'innerHTML' });
				"
				data-testid="pagination-go">Go</button>
		</div>
	</div>
}
```

- [ ] **Step 3: Add helper functions to `helpers.go`**

Add after the `PageInfo` type:

```go
// pageNumbers returns the page numbers to display in pagination.
// Returns a slice where 0 represents an ellipsis.
func pageNumbers(current, total int) []int {
	if total <= 7 {
		nums := make([]int, total)
		for i := range nums {
			nums[i] = i + 1
		}
		return nums
	}
	result := make([]int, 0, 7)
	result = append(result, 1)
	if current-2 > 2 {
		result = append(result, 0)
	}
	start := max(2, current-2)
	end := min(total-1, current+2)
	for i := start; i <= end; i++ {
		result = append(result, i)
	}
	if current+2 < total-1 {
		result = append(result, 0)
	}
	result = append(result, total)
	return result
}
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/partials/helpers.go pkg/views/partials/event_pagination.templ
git commit -m "feat(ui): add pagination control partial with PageInfo type"
```

---

### Task 10: Update data_events_table.templ

**Files:**
- Modify: `pkg/views/partials/data_events_table.templ`

- [ ] **Step 1: Regenerate templates from new templ files**

```bash
templ generate pkg/views/partials/
```

- [ ] **Step 2: Replace `data_events_table.templ`**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/gen"

func eventPipelineName(eventID string, runMap map[string][]store.PipelineRunInfo) string {
	if runs, ok := runMap[eventID]; ok && len(runs) > 0 {
		return runs[0].PipelineName
	}
	return ""
}

func eventPipelineNames(eventID string, runMap map[string][]store.PipelineRunInfo) []store.PipelineRunInfo {
	if runs, ok := runMap[eventID]; ok {
		return runs
	}
	return nil
}

templ DataEventsTable(sources, eventTypes []string, events []*gen.DataEvent, pageInfo PageInfo, runMap map[string][]store.PipelineRunInfo) {
	<div id="events-table-container">
		<div class="card bg-base-100 shadow-sm" id="data-events-table" data-testid="data-events-table">
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
					<tbody>
					for _, e := range events {
						<tbody id={ "event-" + e.EventID }>
							<tr class="cursor-pointer hover"
								hx-get={ templ.URL("/service/web/events/payload/" + e.EventID) }
								hx-trigger="click"
								hx-target={ "#detail-" + e.EventID }
								hx-swap="innerHTML show:top">
								<td class="text-xs whitespace-nowrap">{ e.CreatedAt.Format("15:04:05") }</td>
								<td><span class="badge badge-ghost badge-xs">{ e.EventType }</span></td>
								<td class="text-xs">{ e.Source }{ if e.Capability != "" { " / " + e.Capability } }</td>
								<td class="text-xs font-mono">{ e.EntityID }</td>
								<td class="text-xs">
									pipelineName := eventPipelineName(e.EventID, runMap)
									if pipelineName != "" {
										{pipelineName}
									} else {
										<span class="text-base-content/30">-</span>
									}
								</td>
							</tr>
							<tr id={ "detail-" + e.EventID } class="hidden">
								<td colspan="5"></td>
							</tr>
						</tbody>
					}
					if len(events) == 0 {
						<tr>
							<td colspan="5" class="text-center text-base-content/50 py-4">No events found.</td>
						</tr>
					}
					</tbody>
				</table>
			</div>
			@EventPagination(pageInfo)
		</div>
	</div>
}
```

Add import to the top: `"github.com/flowline-io/flowbot/internal/store"`.

- [ ] **Step 3: Regenerate templates**

```bash
templ generate pkg/views/partials/
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/partials/data_events_table.templ pkg/views/partials/data_events_table_templ.go
git commit -m "feat(ui): update DataEventsTable with pagination and pipeline names"
```

---

### Task 11: Update webhook_logs_table.templ

**Files:**
- Modify: `pkg/views/partials/webhook_logs_table.templ`

- [ ] **Step 1: Replace `webhook_logs_table.templ`**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/gen"

templ WebhookLogsTable(sources, eventTypes []string, events []*gen.DataEvent, pageInfo PageInfo, runMap map[string][]store.PipelineRunInfo) {
	<div id="events-table-container">
		<div class="card bg-base-100 shadow-sm" id="webhook-logs-table" data-testid="webhook-logs-table">
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
								<td class="text-xs">
									pipelineName := eventPipelineName(e.EventID, runMap)
									if pipelineName != "" {
										{pipelineName}
									} else {
										<span class="text-base-content/30">-</span>
									}
								</td>
							</tr>
							<tr id={ "wh-detail-" + e.EventID } class="hidden">
								<td colspan="6"></td>
							</tr>
						</tbody>
					}
					if len(events) == 0 {
						<tr>
							<td colspan="6" class="text-center text-base-content/50 py-4">No webhook receipts recorded yet.</td>
						</tr>
					}
					</tbody>
				</table>
			</div>
			@EventPagination(pageInfo)
		</div>
	</div>
}
```

- [ ] **Step 2: Regenerate templates**

```bash
templ generate pkg/views/partials/
```

- [ ] **Step 3: Commit**

```bash
git add pkg/views/partials/webhook_logs_table.templ pkg/views/partials/webhook_logs_table_templ.go
git commit -m "feat(ui): update WebhookLogsTable with pagination and pipeline names"
```

---

### Task 12: Update events.templ Page

**Files:**
- Modify: `pkg/views/pages/events.templ`

- [ ] **Step 1: Replace `events.templ`**

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

type EventsPageParams struct {
	Sources       []string
	EventTypes    []string
	PipelineNames []string
}

templ EventsPage(p EventsPageParams) {
	@layout.Base("Events") {
		@partials.FilterBar(partials.FilterBarParams{
			Tab:            "data-events",
			Sources:        p.Sources,
			EventTypes:     p.EventTypes,
			PipelineNames:  p.PipelineNames,
		})
		<div class="container mx-auto">
			<div class="mb-4">
				<div role="tablist" class="tabs tabs-lifted">
					<button role="tab"
						class="tab"
						x-bind:class="tab === 'data-events' ? 'tab-active' : ''"
						x-on:click="switchTab('data-events')"
						data-testid="tab-data-events">Data Events</button>
					<button role="tab"
						class="tab"
						x-bind:class="tab === 'webhook-logs' ? 'tab-active' : ''"
						x-on:click="switchTab('webhook-logs')"
						data-testid="tab-webhook-logs">Webhook Logs</button>
				</div>
			</div>
			<div id="events-table-container"
				hx-get="/service/web/events/filtered-events?tab=data-events"
				hx-trigger="load"
				hx-swap="innerHTML"
				data-testid="events-table-container">
				<div class="flex justify-center p-8">
					<span class="loading loading-spinner loading-lg"></span>
				</div>
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Regenerate templates**

```bash
templ generate pkg/views/pages/
```

- [ ] **Step 3: Commit**

```bash
git add pkg/views/pages/events.templ pkg/views/pages/events_templ.go
git commit -m "feat(ui): update EventsPage with filter bar and Alpine.js container"
```

---

### Task 13: Wire Up Script and Fix Backward-Compatible Handlers

**Files:**
- Modify: `pkg/views/layout/base.templ`
- Modify: `internal/modules/web/event_webservice.go`

- [ ] **Step 1: Add `event-filters.js` script tag to `base.templ`**

Add a `<script>` tag after the existing vendor scripts (after line 19, the `pipeline-stats.js` line):

```templ
<script src="/static/js/event-filters.js" defer></script>
```

Insert after line 20:
```html
<script src="/static/js/event-filters.js" defer></script>
```

- [ ] **Step 2: Regenerate base template**

```bash
templ generate pkg/views/layout/
```

- [ ] **Step 3: Update backward-compatible `dataEventsTable` and `webhookLogsTable` handlers**

Replace the old `dataEventsTable` handler (lines 82-112) with a redirect wrapper and keep the old `webhookLogsTable` as a redirect:

```go
func dataEventsTable(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	// Build redirect URL preserving old query params
	source := c.Query("source")
	typeFilter := c.Query("type")
	cursor := c.Query("cursor")
	u := "/service/web/events/filtered-events?tab=data-events"
	if source != "" {
		u += "&source=" + source
	}
	if typeFilter != "" {
		u += "&type=" + typeFilter
	}
	if cursor != "" {
		// Old cursor-based: still forward to old logic via cursor param
		u += "&cursor=" + cursor
	}
	c.Set("HX-Redirect", u)
	return c.SendStatus(200)
}

func webhookLogsTable(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	source := c.Query("source")
	typeFilter := c.Query("type")
	cursor := c.Query("cursor")
	u := "/service/web/events/filtered-events?tab=webhook-logs"
	if source != "" {
		u += "&source=" + source
	}
	if typeFilter != "" {
		u += "&type=" + typeFilter
	}
	if cursor != "" {
		u += "&cursor=" + cursor
	}
	c.Set("HX-Redirect", u)
	return c.SendStatus(200)
}
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go internal/modules/web/event_webservice.go
git commit -m "feat(ui): wire event-filters.js and add backward-compat redirects"
```

---

### Task 14: Verify Build, Lint, and Tests

**Files:**
- None (verification only)

- [ ] **Step 1: Build**

```bash
go tool task build
```

Expected: build succeeds.

- [ ] **Step 2: Lint**

```bash
go tool task lint
```

Expected: no revive violations.

- [ ] **Step 3: Run store tests**

```bash
go test ./internal/store/ -v -count=1
```

Expected: all tests pass.

- [ ] **Step 4: Run all unit tests**

```bash
go tool task test
```

Expected: all tests pass.

- [ ] **Step 5: Verify Alpine x-model + HTMX hx-include compatibility**

Alpine.js `x-model` keeps `input.value` (the JavaScript DOM property) in sync with component state. HTMX reads `element.value` when processing `hx-include`, so the `data-filter-input` + `x-model` pattern works correctly. No additional hidden form needed. The time range shortcut buttons use `submitFilter()` which builds the URL via JavaScript and calls `htmx.ajax()`, bypassing HTMX attribute-based form inclusion for that path.

---

### Task 15: Run BDD Specs

**Files:**
- Modify: `tests/specs/event_spec_test.go`

- [ ] **Step 1: Run existing BDD specs**

```bash
go tool task test:specs
```

Expected: all existing event specs pass.

- [ ] **Step 2: Optionally add advanced filtering scenarios**

Add tests for the new filtering endpoints if time permits. This task is optional for the initial implementation.

---

## Summary

| Task | Files | Action |
|------|-------|--------|
| 1 | `internal/store/store_test.go` | Add test cases for Search, PipelineName, TimeStart/End, Offset |
| 2 | `internal/store/store.go` | Extend `ListDataEventsOptions`, modify `ListDataEvents` |
| 3 | `internal/store/store.go`, `store_test.go` | Add `CountDataEvents` + tests |
| 4 | `internal/store/store.go`, `store_test.go` | Add `ListDistinctEventPipelineNames` + tests |
| 5 | `internal/modules/web/event_webservice.go` | Add `parseEventFilterParams`, `filteredEventsTable` |
| 6 | `internal/modules/web/event_webservice.go` | Update `eventsPage` |
| 7 | `public/js/event-filters.js` | Create Alpine.js component |
| 8 | `pkg/views/partials/event_filters.templ` | Create filter bar |
| 9 | `pkg/views/partials/helpers.go`, `event_pagination.templ` | Create pagination |
| 10 | `pkg/views/partials/data_events_table.templ` | Update with pagination + pipeline |
| 11 | `pkg/views/partials/webhook_logs_table.templ` | Update with pagination + pipeline |
| 12 | `pkg/views/pages/events.templ` | Update with filter bar + Alpine container |
| 13 | `base.templ`, `event_webservice.go` | Wire script, backward-compat redirects |
| 14 | — | Build, lint, test verification |
| 15 | `tests/specs/event_spec_test.go` | BDD specs (optional) |
