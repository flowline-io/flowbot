# Pipeline Run Statistics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add statistical charting (success rate trend, duration distribution, trigger source pie) to Pipeline list page and per-pipeline run history page.

**Architecture:** New `trigger_source` enum on `pipeline_runs`, `PipelineStats()` store method using ent's GroupBy + Aggregate + raw SQL for complex expressions, two API routes serving JSON/HTML, `pipeline_stats.templ` partial with Chart.js canvases lazy-loaded via HTMX, Chart.js added to vendor set.

**Tech Stack:** Ent v0.14, SQL aggregation, Fiber v3, templ, Chart.js v4 CDN, DaisyUI v5, HTMX 2.x

---

### Task 1: Schema — Add trigger_source field to pipeline_runs

**Files:**
- Modify: `internal/store/ent/schema/pipeline_run.go`

- [ ] **Step 1: Add enum field to schema**

In `internal/store/ent/schema/pipeline_run.go`, after line 23 (`field.String("event_type").Default("")`), insert:

```go
field.Enum("trigger_source").
    Values("event", "webhook", "cron", "manual").
    Default("event"),
```

The complete `Fields()` function:

```go
func (PipelineRun) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("id").Immutable(),
        field.String("pipeline_name").NotEmpty(),
        field.String("event_id").NotEmpty().Unique(),
        field.String("event_type").Default(""),
        field.Enum("trigger_source").
            Values("event", "webhook", "cron", "manual").
            Default("event"),
        field.Int("status").Default(0),
        field.String("error").Optional().Default(""),
        field.JSON("checkpoint_data", map[string]any{}).Optional(),
        field.Time("last_heartbeat").Optional().Nillable(),
        field.Time("started_at"),
        field.Time("completed_at").Optional().Nillable(),
        field.Time("created_at").Immutable().Default(time.Now),
    }
}
```

- [ ] **Step 2: Regenerate ent code**

Run: `go tool task ent`
Expected: No errors. Generated files under `internal/store/ent/gen/` updated.

- [ ] **Step 3: Commit**

```bash
git add internal/store/ent/schema/pipeline_run.go internal/store/ent/gen/
git commit -m "feat: add trigger_source enum field to pipeline_runs"
```

---

### Task 2: Types — Define PipelineStats struct

**Files:**
- Create: `pkg/types/pipeline_stats.go`

- [ ] **Step 1: Create types file**

```go
package types

// PipelineStats holds aggregated pipeline run statistics for chart rendering.
type PipelineStats struct {
    SuccessRateTrend    []SuccessRatePoint   `json:"success_rate_trend"`
    DurationDistribution DurationDistribution `json:"duration_distribution"`
    TriggerSourcePie    []TriggerSourceCount  `json:"trigger_source_pie"`
}

// SuccessRatePoint is a single data point on the success rate trend chart.
type SuccessRatePoint struct {
    Date    string  `json:"date"`
    Total   int64   `json:"total"`
    Success int64   `json:"success"`
    Rate    float64 `json:"rate"`
}

// DurationDistribution holds pipeline and step duration bucket counts.
type DurationDistribution struct {
    Pipeline []DurationEntry `json:"pipeline"`
    Step     []DurationEntry `json:"step"`
}

// DurationEntry counts runs that fell into a named duration bucket.
type DurationEntry struct {
    Bucket string `json:"bucket"`
    Count  int64  `json:"count"`
}

// TriggerSourceCount counts pipeline runs grouped by trigger source.
type TriggerSourceCount struct {
    Source string `json:"source"`
    Count  int64  `json:"count"`
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/types/pipeline_stats.go
git commit -m "feat: add PipelineStats types for chart aggregation"
```

---

### Task 3: Store — Add PipelineStats aggregation method with tests

**Files:**
- Modify: `internal/store/store.go`
- Create: `internal/store/store_stats_test.go`

**Implementation approach for PipelineStats:**

The generated ent code provides `PipelineRunGroupBy` with `Scan(ctx, dest)` that scans into arbitrary Go structs. For complex expressions like `DATE_TRUNC` and duration bucketing, use `Modify(func(*sql.Selector))` to inject custom SQL.

Add imports to store.go:
```go
import (
    "entgo.io/ent/dialect/sql"
    "github.com/flowline-io/flowbot/pkg/types"
    // ... existing imports
)
```

- [ ] **Step 1: Add PipelineStats method to PipelineStore in store.go**

Append after the `ListPublishedDefinitions` method (around line 1080):

```go
// PipelineStats returns aggregated pipeline run statistics for chart rendering.
// name empty = all pipelines. since zero = no time filter. groupBy = "day"|"week"|"month".
func (s *PipelineStore) PipelineStats(ctx context.Context, name string, since time.Time, groupBy string) (*types.PipelineStats, error) {
    if s == nil || s.client == nil {
        return emptyPipelineStats(), nil
    }
    stats := &types.PipelineStats{}

    var err error
    stats.SuccessRateTrend, err = s.loadSuccessRate(ctx, name, since, groupBy)
    if err != nil {
        return nil, fmt.Errorf("success rate: %w", err)
    }
    stats.DurationDistribution.Pipeline, err = s.loadDurationBuckets(ctx, name, since)
    if err != nil {
        return nil, fmt.Errorf("pipeline duration: %w", err)
    }
    stats.DurationDistribution.Step, err = s.loadStepDurationBuckets(ctx, name, since)
    if err != nil {
        return nil, fmt.Errorf("step duration: %w", err)
    }
    stats.TriggerSourcePie, err = s.loadTriggerSources(ctx, name, since)
    if err != nil {
        return nil, fmt.Errorf("trigger sources: %w", err)
    }
    return stats, nil
}

// loadSuccessRate runs a GROUP BY date query returning []SuccessRatePoint.
// Uses ent's Modify() to inject DATE() expression.
func (s *PipelineStore) loadSuccessRate(ctx context.Context, name string, since time.Time, groupBy string) ([]types.SuccessRatePoint, error) {
    q := s.client.PipelineRun.Query().Where(pipelinerun.CompletedAtNotNil())
    if name != "" {
        q = q.Where(pipelinerun.PipelineName(name))
    }
    if !since.IsZero() {
        q = q.Where(pipelinerun.StartedAtGTE(since))
    }

    // Determine date trunc expression (SQLite-compatible for tests)
    dateExpr := "DATE(completed_at)"
    if groupBy == "week" {
        dateExpr = "strftime('%Y-W%W', completed_at)"
    } else if groupBy == "month" {
        dateExpr = "strftime('%Y-%m', completed_at)"
    }

    type row struct {
        Date    string `sql:"date"`
        Total   int64  `sql:"total"`
        Success int64  `sql:"success_count"`
    }
    var rows []row

    err := q.Modify(func(sel *sql.Selector) {
        sel.Select(
            sql.Expr(dateExpr + " AS date"),
            sql.Expr("COUNT(*) AS total"),
            sql.Expr("SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) AS success_count"),
        )
        sel.GroupBy(sql.Expr("1"))
        sel.OrderBy(sql.Expr("1"))
    }).Scan(ctx, &rows)

    if err != nil {
        return nil, err
    }

    points := make([]types.SuccessRatePoint, len(rows))
    for i, r := range rows {
        rate := float64(0)
        if r.Total > 0 {
            rate = float64(r.Success) / float64(r.Total)
        }
        points[i] = types.SuccessRatePoint{
            Date: r.Date, Total: r.Total, Success: r.Success, Rate: rate,
        }
    }
    return points, nil
}

// loadDurationBuckets counts runs by duration into 4 buckets: 0-1s, 1-5s, 5-30s, 30s+.
// Uses ent's Modify() + Scan() for aggregation.
func (s *PipelineStore) loadDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
    // Query pipeline_runs for pipeline-level duration
    q := s.client.PipelineRun.Query().Where(pipelinerun.CompletedAtNotNil())
    if name != "" {
        q = q.Where(pipelinerun.PipelineName(name))
    }
    if !since.IsZero() {
        q = q.Where(pipelinerun.StartedAtGTE(since))
    }

    type bucketRow struct {
        Bucket string `sql:"bucket"`
        Count  int64  `sql:"count"`
    }
    var rows []bucketRow

    // SQLite: julianday for test compatibility.
    // In production (Postgres), use EXTRACT(EPOCH FROM ...). Consider dialect detection.
    err := q.Modify(func(sel *sql.Selector) {
        sel.Select(
            sql.Expr(`CASE
                WHEN (julianday(completed_at) - julianday(started_at)) * 86400000 < 1000 THEN '0-1s'
                WHEN (julianday(completed_at) - julianday(started_at)) * 86400000 < 5000 THEN '1-5s'
                WHEN (julianday(completed_at) - julianday(started_at)) * 86400000 < 30000 THEN '5-30s'
                ELSE '30s+'
            END AS bucket`),
            sql.Expr("COUNT(*) AS count"),
        )
        sel.GroupBy(sql.Expr("1"))
        sel.OrderBy(sql.Expr("1"))
    }).Scan(ctx, &rows)

    if err != nil {
        return nil, err
    }

    result := emptyDurationBuckets()
    bucketMap := map[string]int{"0-1s": 0, "1-5s": 1, "5-30s": 2, "30s+": 3}
    for _, r := range rows {
        if idx, ok := bucketMap[r.Bucket]; ok {
            result[idx].Count = r.Count
        }
    }
    return result, nil
}

// loadStepDurationBuckets is the step-level equivalent of loadDurationBuckets.
func (s *PipelineStore) loadStepDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
    q := s.client.PipelineStepRun.Query().Where(pipelinesteprun.CompletedAtNotNil())
    if name != "" {
        q = q.Where(pipelinesteprun.HasPipelineRunWith(pipelinerun.PipelineName(name)))
    }
    if !since.IsZero() {
        q = q.Where(pipelinesteprun.StartedAtGTE(since))
    }

    type bucketRow struct {
        Bucket string `sql:"bucket"`
        Count  int64  `sql:"count"`
    }
    var rows []bucketRow

    err := q.Modify(func(sel *sql.Selector) {
        sel.Select(
            sql.Expr(`CASE
                WHEN (julianday(completed_at) - julianday(started_at)) * 86400000 < 1000 THEN '0-1s'
                WHEN (julianday(completed_at) - julianday(started_at)) * 86400000 < 5000 THEN '1-5s'
                WHEN (julianday(completed_at) - julianday(started_at)) * 86400000 < 30000 THEN '5-30s'
                ELSE '30s+'
            END AS bucket`),
            sql.Expr("COUNT(*) AS count"),
        )
        sel.GroupBy(sql.Expr("1"))
        sel.OrderBy(sql.Expr("1"))
    }).Scan(ctx, &rows)

    if err != nil {
        return nil, err
    }

    result := emptyDurationBuckets()
    bucketMap := map[string]int{"0-1s": 0, "1-5s": 1, "5-30s": 2, "30s+": 3}
    for _, r := range rows {
        if idx, ok := bucketMap[r.Bucket]; ok {
            result[idx].Count = r.Count
        }
    }
    return result, nil
}

// loadTriggerSources counts runs grouped by trigger_source.
func (s *PipelineStore) loadTriggerSources(ctx context.Context, name string, since time.Time) ([]types.TriggerSourceCount, error) {
    q := s.client.PipelineRun.Query()
    if name != "" {
        q = q.Where(pipelinerun.PipelineName(name))
    }
    if !since.IsZero() {
        q = q.Where(pipelinerun.StartedAtGTE(since))
    }

    type row struct {
        Source string `sql:"trigger_source"`
        Count  int64  `sql:"count"`
    }
    var rows []row

    err := q.GroupBy(pipelinerun.FieldTriggerSource).
        Aggregate(gen.Count()).
        Scan(ctx, &rows)

    if err != nil {
        return nil, err
    }

    // All 4 sources must appear (even with count=0)
    result := map[string]int64{"event": 0, "webhook": 0, "cron": 0, "manual": 0}
    for _, r := range rows {
        result[r.Source] = r.Count
    }
    return []types.TriggerSourceCount{
        {Source: "event", Count: result["event"]},
        {Source: "webhook", Count: result["webhook"]},
        {Source: "cron", Count: result["cron"]},
        {Source: "manual", Count: result["manual"]},
    }, nil
}

func emptyPipelineStats() *types.PipelineStats {
    return &types.PipelineStats{
        TriggerSourcePie: []types.TriggerSourceCount{
            {Source: "event"}, {Source: "webhook"}, {Source: "cron"}, {Source: "manual"},
        },
        DurationDistribution: types.DurationDistribution{
            Pipeline: emptyDurationBuckets(),
            Step:     emptyDurationBuckets(),
        },
    }
}

func emptyDurationBuckets() []types.DurationEntry {
    return []types.DurationEntry{
        {Bucket: "0-1s"}, {Bucket: "1-5s"}, {Bucket: "5-30s"}, {Bucket: "30s+"},
    }
}
```

**NOTE:** The SQL expressions use SQLite syntax (`julianday`, `strftime`) for test compatibility. For PostgreSQL in production, these need `EXTRACT(EPOCH FROM ...)` and `DATE_TRUNC(...)`. The implementation should detect the database dialect (e.g., from ent's `drv.Dialect()`) and choose the appropriate SQL expression.

- [ ] **Step 2: Write store test**

Create `internal/store/store_stats_test.go`:

```go
package store

import (
    "context"
    "testing"
    "time"

    "github.com/flowline-io/flowbot/internal/store/ent/schema"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPipelineStats_SuccessRateTrend(t *testing.T) {
    client := getTestClient(t)
    s := NewPipelineStore(client)
    ctx := context.Background()
    now := time.Now()

    // Create runs with different statuses and trigger sources
    sources := []string{"event", "event", "webhook", "cron"}
    statuses := []int{int(schema.PipelineDone), int(schema.PipelineDone), int(schema.PipelineCancel), int(schema.PipelineDone)}
    for i, src := range sources {
        run, err := s.CreateRun(ctx, "s1", "eid-"+src, "t.evt", src)
        require.NoError(t, err)
        _, err = client.PipelineRun.UpdateOneID(run.ID).
            SetStatus(statuses[i]).
            SetCompletedAt(now).
            Save(ctx)
        require.NoError(t, err)
    }

    tests := []struct {
        name    string
        pName   string
        since   time.Time
        groupBy string
        minRows int
    }{
        {name: "global stats no time filter", pName: "", since: time.Time{}, groupBy: "day", minRows: 1},
        {name: "single pipeline", pName: "s1", since: time.Time{}, groupBy: "day", minRows: 1},
        {name: "future since returns empty", pName: "s1", since: now.Add(24 * time.Hour), groupBy: "day", minRows: 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            stats, err := s.PipelineStats(ctx, tt.pName, tt.since, tt.groupBy)
            require.NoError(t, err)
            require.NotNil(t, stats)
            assert.GreaterOrEqual(t, len(stats.SuccessRateTrend), tt.minRows)
            assert.Len(t, stats.TriggerSourcePie, 4)
            // Verify all 4 sources present
            srcCount := make(map[string]bool)
            for _, sc := range stats.TriggerSourcePie {
                srcCount[sc.Source] = true
            }
            assert.True(t, srcCount["event"])
            assert.True(t, srcCount["webhook"])
            assert.True(t, srcCount["cron"])
            assert.True(t, srcCount["manual"])
            // Duration buckets exist
            assert.Len(t, stats.DurationDistribution.Pipeline, 4)
            assert.Len(t, stats.DurationDistribution.Step, 4)
        })
    }
}

func TestPipelineStats_NilSafe(t *testing.T) {
    var nilStore *PipelineStore
    stats, err := nilStore.PipelineStats(context.Background(), "", time.Time{}, "day")
    require.NoError(t, err)
    require.NotNil(t, stats)
    assert.Len(t, stats.TriggerSourcePie, 4)
}

func TestPipelineStats_ZeroValueClient(t *testing.T) {
    store := &PipelineStore{client: nil}
    stats, err := store.PipelineStats(context.Background(), "", time.Time{}, "day")
    require.NoError(t, err)
    require.NotNil(t, stats)
    assert.Len(t, stats.TriggerSourcePie, 4)
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/store/ -run TestPipelineStats -count=1 -v`
Expected: Tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/store/store.go internal/store/store_stats_test.go
git commit -m "feat: add PipelineStats aggregation method with tests"
```

---

### Task 4: Engine & Store — Thread triggerSource through CreateRun

**Files:**
- Modify: `pkg/pipeline/engine.go:72` (RunStore interface)
- Modify: `internal/store/store.go:680` (CreateRun signature + body)
- Modify: `pkg/pipeline/engine.go:397-405` (createRunRecord)
- Modify: `pkg/pipeline/engine.go:178` (executePipeline — add triggerSource param)
- Modify: `pkg/pipeline/engine.go:169` (handleEvent — pass "event")
- Modify: `pkg/pipeline/engine.go:661` (executeCronJob — pass "cron")
- Modify: `pkg/pipeline/engine.go:705` (ExecuteWebhook — pass "webhook")
- Modify: `pkg/pipeline/engine.go:442-460` (ResumePipeline — find createRunRecord call)
- Modify: `internal/store/store_test.go` (add "event" to existing CreateRun calls)
- Modify: `tests/specs/event_spec_test.go` (add "event" to existing CreateRun calls)

- [ ] **Step 1: Update RunStore interface**

In `pkg/pipeline/engine.go:72`, change:
```go
CreateRun(ctx context.Context, pipelineName, eventID, eventType string) (*gen.PipelineRun, error)
```
to:
```go
CreateRun(ctx context.Context, pipelineName, eventID, eventType, triggerSource string) (*gen.PipelineRun, error)
```

- [ ] **Step 2: Update PipelineStore.CreateRun**

In `internal/store/store.go:680`, change signature to add `triggerSource string` param, and add `.SetTriggerSource(triggerSource)` to the builder chain.

- [ ] **Step 3: Update createRunRecord**

In `pkg/pipeline/engine.go:397`, add `triggerSource string` param, pass through to `e.store.CreateRun()`.

- [ ] **Step 4: Thread triggerSource through executePipeline**

Add `triggerSource string` param to `executePipeline`. Update:
- `handleEvent` → pass `"event"`
- `executeCronJob` → generate synthetic event, pass `"cron"` to `executePipeline`
- `ExecuteWebhook` → pass `"webhook"` to `executePipeline`
- Any `ResumePipeline` call that calls `createRunRecord`

- [ ] **Step 5: Update all test and spec CreateRun callers**

Search for all `CreateRun(` calls in test files. Use: `rg "CreateRun\(" --include="*.go" -l`. Add `"event"` as the 5th argument.

- [ ] **Step 6: Verify build and tests**

Run: `go build ./...`
Run: `go test ./internal/store/ ./pkg/pipeline/ ./tests/specs/ -count=1`
Expected: All compile and tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/store/store.go pkg/pipeline/engine.go internal/server/webhook.go internal/store/store_test.go tests/specs/event_spec_test.go
git commit -m "feat: thread trigger_source through pipeline CreateRun"
```

---

### Task 5: Store — Add TriggerSource to PipelineRunInfo

**Files:**
- Modify: `internal/store/store.go` (PipelineRunInfo struct + GetPipelineRunsForEvents)

- [ ] **Step 1: Add field to PipelineRunInfo**

Find `PipelineRunInfo` struct in store.go around line 448. Add:
```go
TriggerSource string
```

- [ ] **Step 2: Update GetPipelineRunsForEvents query**

In the query around line 649, add `pipelinerun.TriggerSource` to the selection (or map from the returned run entity).

- [ ] **Step 3: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: expose trigger_source in PipelineRunInfo"
```

---

### Task 6: API — Add stats routes

**Files:**
- Modify: `internal/modules/web/pipeline_webservice.go`

- [ ] **Step 1: Add routes**

Add to `pipelineWebserviceRules` (after line 44):
```go
webservice.Get("/pipelines/stats", pipelineStats),
webservice.Get("/pipelines/:name/stats", pipelineStats),
```

- [ ] **Step 2: Implement pipelineStats handler**

```go
func pipelineStats(c fiber.Ctx) error {
    name := c.Params("name") // empty for global
    sinceStr := c.Query("since", "")
    since := time.Time{}
    if sinceStr != "" {
        parsed, err := time.Parse("2006-01-02", sinceStr)
        if err != nil {
            return types.Errorf(types.ErrInvalidArgument, "invalid since date: %v", err)
        }
        since = parsed
    } else {
        since = time.Now().AddDate(0, 0, -30)
    }
    groupBy := c.Query("groupBy", "day")
    if groupBy != "day" && groupBy != "week" && groupBy != "month" {
        return types.Errorf(types.ErrInvalidArgument, "groupBy must be day, week, or month")
    }

    s := getPipelineDefStore()
    if s == nil {
        return types.Errorf(types.ErrInternal, "store not available")
    }
    if name != "" {
        _, err := s.GetDefinitionByName(context.Background(), name)
        if err != nil {
            if errors.Is(err, types.ErrNotFound) {
                return types.Errorf(types.ErrNotFound, "pipeline %s not found", name)
            }
            return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
        }
    }

    stats, err := s.PipelineStats(context.Background(), name, since, groupBy)
    if err != nil {
        return types.Errorf(types.ErrInternal, "pipeline stats: %v", err)
    }

    if c.Get("Accept", "") == "application/json" {
        return c.JSON(stats)
    }

    c.Type("html")
    return partials.PipelineStats(name, stats).Render(context.Background(), c.Response().BodyWriter())
}
```

Note: the partial signature is `partials.PipelineStats(name, stats)` — no `since`/`groupBy` in the template call since the time range/group-by tabs are built into the partial and self-contained via HTMX links.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go
git commit -m "feat: add GET /pipelines/stats and /pipelines/:name/stats routes"
```

---

### Task 7: Views — Create pipeline_stats.templ partial

**Files:**
- Create: `pkg/views/partials/pipeline_stats.templ`

- [ ] **Step 1: Create the partial**

```go
package partials

import (
    "time"

    "github.com/bytedance/sonic"
    "github.com/flowline-io/flowbot/pkg/types"
)

templ PipelineStats(name string, stats *types.PipelineStats) {
    <div id="pipeline-stats-container" data-testid="pipeline-stats-container">
        <!-- Time Range & Group-by Tabs -->
        <div class="flex items-center gap-3 mb-4 flex-wrap">
            <div class="join" data-testid="time-range-tabs">
                <button type="button"
                    class="join-item btn btn-sm btn-active"
                    hx-get={ buildStatsURL(name, 30, "day") }
                    hx-target="#pipeline-stats-container"
                    hx-swap="outerHTML"
                    data-testid="btn-range-30d">30d</button>
                <button type="button"
                    class="join-item btn btn-sm"
                    hx-get={ buildStatsURL(name, 90, "day") }
                    hx-target="#pipeline-stats-container"
                    hx-swap="outerHTML"
                    data-testid="btn-range-90d">90d</button>
                <button type="button"
                    class="join-item btn btn-sm"
                    hx-get={ buildStatsURL(name, 0, "day") }
                    hx-target="#pipeline-stats-container"
                    hx-swap="outerHTML"
                    data-testid="btn-range-all">All</button>
            </div>
            <div class="join" data-testid="groupby-tabs">
                <button type="button"
                    class="join-item btn btn-sm btn-active"
                    hx-get={ buildStatsURL(name, 30, "day") }
                    hx-target="#pipeline-stats-container"
                    hx-swap="outerHTML"
                    data-testid="btn-groupby-day">day</button>
                <button type="button"
                    class="join-item btn btn-sm"
                    hx-get={ buildStatsURL(name, 30, "week") }
                    hx-target="#pipeline-stats-container"
                    hx-swap="outerHTML"
                    data-testid="btn-groupby-week">week</button>
                <button type="button"
                    class="join-item btn btn-sm"
                    hx-get={ buildStatsURL(name, 30, "month") }
                    hx-target="#pipeline-stats-container"
                    hx-swap="outerHTML"
                    data-testid="btn-groupby-month">month</button>
            </div>
        </div>

        <!-- Charts -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
            <div class="lg:col-span-2 card bg-base-100 shadow-sm" data-testid="chart-success-rate">
                <div class="card-body p-4">
                    <h3 class="card-title text-sm text-base-content/70">Success Rate Trend</h3>
                    <canvas id="chart-success-rate"
                        data-stats={ toJSON(stats) }
                        data-chart-type="line"
                        class="w-full h-64"></canvas>
                </div>
            </div>
            <div class="card bg-base-100 shadow-sm" data-testid="chart-duration">
                <div class="card-body p-4">
                    <h3 class="card-title text-sm text-base-content/70">Duration Distribution</h3>
                    <canvas id="chart-duration"
                        data-chart-type="bar"
                        class="w-full h-64"></canvas>
                </div>
            </div>
            <div class="card bg-base-100 shadow-sm" data-testid="chart-trigger">
                <div class="card-body p-4">
                    <h3 class="card-title text-sm text-base-content/70">Trigger Sources</h3>
                    <canvas id="chart-trigger"
                        data-chart-type="doughnut"
                        class="w-full h-64"></canvas>
                </div>
            </div>
        </div>
    </div>
}

func toJSON(v any) string {
    b, _ := sonic.Marshal(v)
    return string(b)
}

func buildStatsURL(name string, days int, groupBy string) string {
    since := ""
    if days > 0 {
        since = time.Now().AddDate(0, 0, -days).Format("2006-01-02")
    }
    u := "/service/web/pipelines/stats"
    if name != "" {
        u = "/service/web/pipelines/" + name + "/stats"
    }
    if since != "" {
        return u + "?groupBy=" + groupBy + "&since=" + since
    }
    return u + "?groupBy=" + groupBy
}
```

Run: `templ generate pkg/views/partials/pipeline_stats.templ`

- [ ] **Step 2: Commit**

```bash
git add pkg/views/partials/pipeline_stats.templ pkg/views/partials/pipeline_stats_templ.go
git commit -m "feat: add pipeline stats chart partial with HTMX controls"
```

---

### Task 8: Views — Add stats section to pages

**Files:**
- Modify: `pkg/views/pages/pipeline_list.templ`
- Modify: `pkg/views/pages/pipeline_runs.templ`

- [ ] **Step 1: Add to pipeline_list.templ**

Insert before `<div id="pipeline-list-container">` (around line 21):

```templ
<div hx-get="/service/web/pipelines/stats?groupBy=day" hx-trigger="revealed" hx-swap="outerHTML">
    <div class="card bg-base-100 shadow-sm mb-6 animate-pulse">
        <div class="card-body p-6">
            <div class="h-64 bg-base-200 rounded"></div>
        </div>
    </div>
</div>
```

- [ ] **Step 2: Add to pipeline_runs.templ**

Insert before `<div id="pipeline-runs-container">` (around line 17):

```templ
<div hx-get={ templ.URL("/service/web/pipelines/" + name + "/stats?groupBy=day") } hx-trigger="revealed" hx-swap="outerHTML">
    <div class="card bg-base-100 shadow-sm mb-6 animate-pulse">
        <div class="card-body p-6">
            <div class="h-64 bg-base-200 rounded"></div>
        </div>
    </div>
</div>
```

- [ ] **Step 3: Regenerate templ**

Run: `templ generate pkg/views/pages/...`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/views/pages/pipeline_list.templ pkg/views/pages/pipeline_runs.templ pkg/views/pages/*_templ.go
git commit -m "feat: add stats section to pipeline list and runs pages"
```

---

### Task 9: Assets — Chart.js vendor and JS init script

**Files:**
- Create: `public/vendor/chart.js.min.js`
- Create: `public/js/pipeline-stats.js`
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Download Chart.js v4**

```bash
wget -O public/vendor/chart.js.min.js https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js
```

- [ ] **Step 2: Create pipeline-stats.js**

```javascript
(function() {
  'use strict';

  var colors = {
    primary: getComputedStyle(document.documentElement).getPropertyValue('--p') || '#3b82f6',
    success: '#22c55e',
    error: '#ef4444',
    warning: '#f59e0b',
    info: '#06b6d4',
  };

  function initChart(canvas) {
    var type = canvas.dataset.chartType;
    if (!type || canvas._chart) return;

    var statsEl = document.getElementById('chart-success-rate');
    if (!statsEl || !statsEl.dataset.stats) return;

    var stats;
    try { stats = JSON.parse(statsEl.dataset.stats); } catch(e) { return; }

    if (type === 'line') {
      var trend = stats.success_rate_trend || [];
      canvas._chart = new Chart(canvas, {
        type: 'line',
        data: {
          labels: trend.map(function(p) { return p.date; }),
          datasets: [{
            label: 'Success Rate',
            data: trend.map(function(p) { return +(p.rate * 100).toFixed(1); }),
            borderColor: colors.success,
            backgroundColor: colors.success + '20',
            fill: true, tension: 0.2, pointRadius: 3,
          }]
        },
        options: {
          responsive: true, maintainAspectRatio: false,
          scales: { y: { min: 0, max: 100, ticks: { callback: function(v) { return v + '%'; } } } },
          plugins: { legend: { display: false } }
        }
      });
    } else if (type === 'bar') {
      var pipeline = (stats.duration_distribution || {}).pipeline || [];
      canvas._chart = new Chart(canvas, {
        type: 'bar',
        data: {
          labels: pipeline.map(function(b) { return b.bucket; }),
          datasets: [{
            label: 'Pipeline Runs',
            data: pipeline.map(function(b) { return b.count; }),
            backgroundColor: colors.primary,
          }]
        },
        options: {
          responsive: true, maintainAspectRatio: false,
          plugins: { legend: { display: false } },
          scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } }
        }
      });
    } else if (type === 'doughnut') {
      var pie = stats.trigger_source_pie || [];
      canvas._chart = new Chart(canvas, {
        type: 'doughnut',
        data: {
          labels: pie.map(function(s) {
            return s.source.charAt(0).toUpperCase() + s.source.slice(1);
          }),
          datasets: [{
            data: pie.map(function(s) { return s.count; }),
            backgroundColor: [colors.primary, colors.success, colors.warning, colors.info],
          }]
        },
        options: { responsive: true, maintainAspectRatio: false }
      });
    }
  }

  function destroyCharts(container) {
    container.querySelectorAll('canvas').forEach(function(c) {
      if (c._chart) { c._chart.destroy(); c._chart = null; }
    });
  }

  function initAll() {
    document.querySelectorAll('#pipeline-stats-container canvas[data-chart-type]').forEach(initChart);
  }

  document.addEventListener('htmx:beforeSwap', function(evt) {
    if (evt.detail.target.id === 'pipeline-stats-container') {
      destroyCharts(evt.detail.target);
    }
  });

  document.addEventListener('htmx:afterSettle', function(evt) {
    var container = document.getElementById('pipeline-stats-container');
    if (container) initAll();
  });

  document.addEventListener('DOMContentLoaded', initAll);
})();
```

- [ ] **Step 3: Add Chart.js <script> to base layout**

In `pkg/views/layout/base.templ`, after line 18 (`<script src="/static/js/app.js"></script>`:

```html
<script src="/static/vendor/chart.js.min.js" defer></script>
<script src="/static/js/pipeline-stats.js" defer></script>
```

Run: `templ generate pkg/views/layout/base.templ`

- [ ] **Step 4: Commit**

```bash
git add public/vendor/chart.js.min.js public/js/pipeline-stats.js pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: add Chart.js vendor lib and pipeline stats JS init"
```

---

### Task 10: Format, Lint, Test

**Files:** All modified files

- [ ] **Step 1: Format**

Run: `go tool task format`
Expected: Clean or auto-fixed.

- [ ] **Step 2: Lint**

Run: `go tool task lint`
Expected: No errors.

- [ ] **Step 3: Unit tests**

Run: `go test ./internal/store/ ./internal/modules/web/ ./pkg/pipeline/ -count=1`
Expected: All pass.

- [ ] **Step 4: BDD specs**

Run: `go tool task test:specs`
Expected: Pass.

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "chore: format, lint fixes, and final verifications"
```
