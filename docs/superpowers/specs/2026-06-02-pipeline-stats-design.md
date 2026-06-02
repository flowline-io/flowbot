# Pipeline Run Statistics

## Overview

Add statistical charting to the Pipeline list page and per-pipeline run history page. Three chart types display aggregated run data: success rate trend, execution duration distribution, and trigger source breakdown.

## Motivation

The current pipeline UI shows only raw data tables (pipeline definitions, run history). Users have no visibility into pipeline health trends, performance patterns, or trigger usage distribution without querying Prometheus or inspecting individual runs.

## Design

### 1. Database Schema Change

Add `trigger_source` enum field to `pipeline_runs` table (ent schema `internal/store/ent/schema/pipeline_runs.go`):

```go
field.Enum("trigger_source").
    Values("event", "webhook", "cron", "manual").
    Default("event")
```

`PipelineRunInfo` struct in store.go (used by `EventStore.GetPipelineRunsForEvents`) gains `TriggerSource string`.

All existing `CreateRun()` call sites are updated to pass the correct trigger source:
- Engine event handler -> `"event"`
- Webhook handler -> `"webhook"`
- Cron scheduler -> `"cron"`
- Manual test execution -> `"manual"`

### 2. Store Layer

New method on `PipelineStore` in `internal/store/store.go`:

```go
func (s *PipelineStore) PipelineStats(ctx context.Context, name string, since time.Time, groupBy string) (*types.PipelineStats, error)
```

Parameters:
- `name`: empty string = all pipelines; non-empty = single pipeline
- `since`: zero value = no time filter
- `groupBy`: `"day"`, `"week"`, or `"month"`

Returns `types.PipelineStats` defined in `pkg/types/pipeline_stats.go`:

```go
type PipelineStats struct {
    SuccessRateTrend    []SuccessRatePoint `json:"success_rate_trend"`
    DurationDistribution DurationBucket    `json:"duration_distribution"`
    TriggerSourcePie    []TriggerSourceCount `json:"trigger_source_pie"`
}

type SuccessRatePoint struct {
    Date    string  `json:"date"`
    Total   int64   `json:"total"`
    Success int64   `json:"success"`
    Rate    float64 `json:"rate"`
}

type DurationBucket struct {
    Pipeline []DurationEntry `json:"pipeline"`
    Step     []DurationEntry `json:"step"`
}

type DurationEntry struct {
    Bucket string `json:"bucket"` // "0-1s", "1-5s", "5-30s", "30s+"
    Count  int64  `json:"count"`
}

type TriggerSourceCount struct {
    Source string `json:"source"` // "event", "webhook", "cron", "manual"
    Count  int64  `json:"count"`
}
```

SQL implementation uses `DATE_TRUNC` for time bucketing, `EXTRACT(EPOCH FROM ...)` for duration ranges, and `GROUP BY` aggregation. Runs with `completed_at IS NULL` are excluded from success rate and duration calculations.

### 3. API Layer

Two new routes in `internal/modules/web/pipeline_webservice.go`:

```
GET /pipelines/stats?since=2026-05-01&groupBy=day
GET /pipelines/:name/stats?since=2026-05-01&groupBy=day
```

Query parameters:
- `since`: RFC3339 date string, defaults to 30 days ago if missing
- `groupBy`: `"day"` | `"week"` | `"month"`, defaults to `"day"`

Content negotiation:
- `Accept: text/html` -> returns templ-rendered chart HTML fragment (HTMX partial)
- `Accept: application/json` -> returns JSON (Chart.js canvas fetches data via JS)
- Default (missing or `*/*`): returns HTML (HTMX partial), consistent with the page-oriented nature of the existing UI

Both handlers delegate to `store.PipelineStats()`, then render JSON or templ based on Accept header. Error handling follows the project convention: `%w` for wrapping, appropriate HTTP status codes (400 for invalid params, 404 for unknown pipeline name, 500 for DB errors).

### 4. Frontend Layer

#### New files

`pkg/views/partials/pipeline_stats.templ` — chart rendering partial:

- Time range selector: DaisyUI `btn-group` with 7d / 30d / 90d / all options; button clicks trigger HTMX `hx-get` on the container, replacing the entire chart block with fresh data.
- Group-by toggle: adjacent btn-group for day / week / month.
- Three `<canvas>` elements wrapped in `.stats-card` containers with `hx-get` + `hx-trigger="load"` for lazy loading.
- Initial load shows DaisyUI skeleton placeholders before charts render.
- Inline `<script>` block generates Chart.js configurations from the JSON data (fetched via the canvas element's `data-stats-url` attribute).

Layout: wide screens (>1024px) show success rate trend full-width on top, duration distribution and trigger source side-by-side below. Narrow screens stack all three vertically.

#### Modified files

`pkg/views/pages/pipeline_list.templ`:
- Insert stats section ABOVE the pipeline definition table
- Initial render uses `hx-get="/pipelines/stats"` with `hx-trigger="revealed"` and skeleton placeholder

`pkg/views/pages/pipeline_runs.templ`:
- Insert stats section ABOVE the run history table
- Initial render uses `hx-get="/pipelines/:name/stats"` with `hx-trigger="revealed"` and skeleton placeholder

#### Chart.js integration

Add Chart.js to `public/vendor/chart.js/` (CDN download). Reference in the layout template via `<script>` tag. Chart types used: `line` for success rate, `bar` for duration, `doughnut` for trigger source. Colors use DaisyUI theme CSS variables (`oklch(var(--s))`, etc.) for dark mode compatibility.

### 5. Trigger Source Propagation

All call sites of `RunStore.CreateRun()` must pass the correct trigger source. Callers:

| Caller | File | Source |
|--------|------|--------|
| Event handler | `pkg/pipeline/engine.go` (event consumer) | `"event"` |
| Webhook executor | `internal/server/pipeline.go` (webhook routes) | `"webhook"` |
| Cron scheduler | `pkg/pipeline/engine.go` (cron ticker) | `"cron"` |
| Manual test | `internal/modules/web/pipeline_webservice.go` (test handler) | `"manual"` |

`RunStore` interface in `pkg/pipeline/engine.go` gains a new `CreateRun` signature:

```go
CreateRun(ctx context.Context, pipelineName, eventID, eventType, triggerSource string) (int64, error)
```

### 6. Testing

- **Unit tests**: New table-driven tests for `PipelineStats` store method (at least 3 cases: global, single pipeline, time-filtered). Tests for stats handler with mocked store.
- **BDD specs**: New Ginkgo spec covering the stats API endpoint and HTML fragment rendering. Chart rendering is verified by checking for `<canvas>` elements and data attributes in the response, not visual output.

## References

- Existing store pattern: `internal/store/store.go` (PipelineStore section, line ~670)
- Existing webservice pattern: `internal/modules/web/pipeline_webservice.go`
- Existing templ partials: `pkg/views/partials/`
- Existing HTMX lazy loading pattern: search for `hx-trigger="revealed"` in `pkg/views/`
- Pipeline runs schema: `internal/store/ent/schema/pipeline_runs.go`
