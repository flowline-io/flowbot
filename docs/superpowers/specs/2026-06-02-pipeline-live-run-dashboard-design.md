# Pipeline Live Run Dashboard

**Date:** 2026-06-02
**Status:** Approved
**Scope:** Backend + Frontend — engine callbacks, Redis Stream progress events, SSE endpoint, Alpine.js live dashboard page

## Problem

The Run History page (`/pipelines/:name/runs`) is static — users must manually refresh to see step progress. A running pipeline shows no real-time feedback. Operators have no way to watch a pipeline execute step-by-step with live input/output data.

## Design

### 1. Engine Callbacks

A new `StepCallback` interface is injected into the pipeline `Engine`. The engine calls hooks at run and step boundaries. When the callback is `nil` (tests, no Redis), the engine skips calls — existing behavior is unchanged.

**New file: `pkg/pipeline/progress.go`**

```go
type StepCallback interface {
    OnRunStart(ctx context.Context, runID int64, pipelineName string,
        trigger string, totalSteps int, stepNames []string)
    OnStepStart(ctx context.Context, runID int64, pipelineName string,
        stepIndex int, stepName string, input map[string]any)
    OnStepDone(ctx context.Context, runID int64, pipelineName string,
        stepIndex int, stepName string, output map[string]any, elapsedMs int64)
    OnStepError(ctx context.Context, runID int64, pipelineName string,
        stepIndex int, stepName string, err error, elapsedMs int64)
    OnRunComplete(ctx context.Context, runID int64, pipelineName string,
        elapsedMs int64, failed bool, errMsg string)
}

type StepProgressEvent struct {
    RunID        int64          `json:"run_id"`
    PipelineName string         `json:"pipeline_name"`
    StepIndex    int            `json:"step_index"`          // -1 for run-level events
    StepName     string         `json:"step_name"`
    Status       string         `json:"status"`              // "start" | "running" | "done" | "error" | "complete" | "failed"
    Input        map[string]any `json:"input,omitempty"`
    Output       map[string]any `json:"output,omitempty"`
    ElapsedMs    int64          `json:"elapsed_ms,omitempty"`
    Error        string         `json:"error,omitempty"`
    TotalSteps   int            `json:"total_steps,omitempty"` // sent on "start"
}
```

**Call sites in `pkg/pipeline/engine.go`**:

| Hook | Location | When |
|---|---|---|
| `OnRunStart` | `executePipeline()`, before step loop | After run record created |
| `OnStepStart` | `executeStep()`, before `ability.Invoke()` | After params rendered |
| `OnStepDone` | `executeStep()`, after successful invoke | After step record updated |
| `OnStepError` | `executeStep()`, on invoke failure | After error recorded |
| `OnRunComplete` | `executePipeline()`, after `finishRunRecord()` | Run done or failed |

`OnRunComplete` receives the final elapsed time. If any step failed, `failed=true` and `errMsg` contains the last error.

**Trigger format**: The `trigger` parameter in `OnRunStart` is a human-readable description string, matching the existing `Trigger` struct rendering (e.g. `"event:item.created"`, `"webhook:/github-push"`, `"cron:*/5 * * * *"`).

### 2. Redis Stream Publisher

Uses `go-redis` raw `XAdd`/`Expire` commands (not Watermill) — because stream names are dynamic per-run (`pipeline:run:{runID}`) and Watermill requires statically-registered topics.

**New type in `internal/server/pipeline.go`: `pipelineStepCallback`**

```go
type pipelineStepCallback struct {
    rdb *redis.Client
}

func (c *pipelineStepCallback) OnRunStart(ctx, runID, ...) {
    evt := StepProgressEvent{..., Status: "start", StepIndex: -1, TotalSteps: total}
    c.publish(runID, evt)
    c.rdb.Expire(ctx, streamName(runID), 24*time.Hour) // failsafe TTL
}

func (c *pipelineStepCallback) OnStepStart/Done/Error(...) { ... c.publish(runID, evt) }

func (c *pipelineStepCallback) OnRunComplete(ctx, runID, name string, elapsed int64, failed bool, errMsg string) {
    status := "complete"
    if failed { status = "failed" }
    evt := StepProgressEvent{..., Status: status, ElapsedMs: elapsed, Error: errMsg, StepIndex: -1}
    c.publish(runID, evt)
    c.rdb.Expire(ctx, streamName(runID), 5*time.Minute) // drain TTL
}

// publish sends a progress event to Redis Stream asynchronously
// to avoid blocking the pipeline engine on Redis latency.
func (c *pipelineStepCallback) publish(runID int64, evt StepProgressEvent) {
    payload, _ := sonic.Marshal(evt)
    go func() {
        pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        c.rdb.XAdd(pubCtx, &redis.XAddArgs{
            Stream: streamName(runID),
            Values: map[string]any{"data": payload},
        })
    }()
}

func streamName(runID int64) string { return fmt.Sprintf("pipeline:run:%d", runID) }
```

**DI wiring** (`internal/server/fx.go`): Provider `NewPipelineStepCallback(rdb)` supplies `pipeline.StepCallback`, consumed by `Engine` constructor.

### 3. SSE Endpoint

**Route**: `GET /service/web/pipelines/:name/runs/:runID/live/watch`

Authentication via existing cookie auth (same origin, `EventSource` auto-sends cookies). Uses `c.Context().SetBodyStreamWriter()` for reliable flush/lifecycle management.

**Handler in `internal/modules/web/pipeline_webservice.go`**:

```go
func (h moduleHandler) watchPipelineRunLive(c *fiber.Ctx) error {
    runID := c.Params("runID")
    stream := fmt.Sprintf("pipeline:run:%s", runID)

    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")

    ctx := c.Context()
    rdb := rdb.Client // package-level from pkg/rdb

    c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
        lastID := "0"
        for {
            select {
            case <-ctx.Done():
                return
            default:
                result, err := rdb.XRead(ctx, &redis.XReadArgs{
                    Streams: []string{stream, lastID},
                    Count:   10,
                    Block:   5 * time.Second,
                }).Result()

                if errors.Is(err, context.Canceled) {
                    return // client disconnected
                }
                if err == redis.Nil || len(result) == 0 {
                    // timeout waiting for new messages, send heartbeat
                    fmt.Fprintf(w, ": heartbeat\n\n")
                    w.Flush()
                    continue
                }
                if err != nil {
                    // real Redis error, backoff to avoid hot loop
                    time.Sleep(2 * time.Second)
                    continue
                }
                for _, msg := range result[0].Messages {
                    lastID = msg.ID
                    data := msg.Values["data"].(string)
                    fmt.Fprintf(w, "data: %s\n\n", data)
                    w.Flush()
                    if strings.Contains(data, `"status":"complete"`) ||
                       strings.Contains(data, `"status":"failed"`) {
                        return
                    }
                }
            }
        }
    })
    return nil
}
```

### 4. Live Dashboard Page

**Route**: `GET /service/web/pipelines/:name/runs/:runID/live`

Queries PostgreSQL for the run and all existing step runs, pre-renders initial state as Alpine.js data. This ensures the page shows correct state even for partially-complete runs (e.g., 3/5 steps already done).

**New template: `pkg/views/pages/pipeline_run_live.templ`**

**New JS component: `public/js/pipeline-run-live.js`**


### 5. Frontend Component

```js
Alpine.data('pipelineRunLive', (initial) => ({
    runID: initial.runID,
    pipelineName: initial.pipelineName,
    trigger: initial.trigger,
    totalSteps: initial.totalSteps,
    steps: initial.steps,  // [{name, status, elapsed_ms, output, error}]
    selectedIndex: -1,
    totalElapsed: 0,
    completed: 0,
    failedSteps: 0,
    runStatus: initial.runStatus,
    eventSource: null,

    init() {
        this.recalc()

        const idx = this.steps.findIndex(s => s.status === 'running' || s.status === 'pending')
        this.selectedIndex = idx >= 0 ? idx : this.steps.length - 1

        if (this.runStatus === 'running') {
            const watchURL = window.location.pathname.replace(/\/live$/, '/live/watch')
            this.eventSource = new EventSource(watchURL)
            this.eventSource.onmessage = (e) => {
                const evt = JSON.parse(e.data)
                this.applyEvent(evt)
            }
            this.eventSource.onerror = () => {
                if (this.runStatus === 'done' || this.runStatus === 'failed') {
                    this.eventSource.close()
                }
                // otherwise let browser auto-reconnect
            }
        }
    },

    // recalc recomputes summary counters from the steps array.
    // Called on init and after each applyEvent to ensure idempotency.
    recalc() {
        this.completed = this.steps.filter(s => s.status === 'done').length
        this.failedSteps = this.steps.filter(s => s.status === 'error').length
        this.totalElapsed = this.steps.reduce((s, v) => s + (v.elapsed_ms || 0), 0)
    },

    applyEvent(evt) {
        if (evt.step_index === -1) {
            if (evt.status === 'start') this.runStatus = 'running'
            if (evt.status === 'complete') this.runStatus = 'done'
            if (evt.status === 'failed') this.runStatus = 'failed'
            if (evt.elapsed_ms) this.totalElapsed = evt.elapsed_ms
            return
        }
        const step = this.steps[evt.step_index]
        step.status = evt.status
        if (evt.status === 'done') {
            step.output = evt.output; step.elapsed_ms = evt.elapsed_ms
        }
        if (evt.status === 'error') {
            step.error = evt.error; step.elapsed_ms = evt.elapsed_ms
        }
        if (evt.status === 'running') {
            step.input = evt.input
            this.selectedIndex = evt.step_index
        }
        this.recalc()
    },

    selectStep(idx) { this.selectedIndex = idx },
    get selectedStep() { return this.steps[this.selectedIndex] || null }
}))
```

### 6. Page Layout

```
┌──────────────────────────────────────────────────┐
│  Live: sync-issues                       3.2s    │
│  Trigger: github.issue_created                    │
│  Status: ● Running                               │
├──────────────────┬───────────────────────────────┤
│                  │ Status: Running                │
│  ◉ fetch_issue    │ Input: {"repo":"bot",...}      │
│  ○ parse_data     │ Elapsed: 1.2s                 │
│  ○ create_task    │                               │
│  ○ send_notify    │ (Output pending...)           │
│  ○ update_memo    │ Error: none                   │
│                  │                               │
├──────────────────┴───────────────────────────────┤
│  ✓ 2  ◉ 1  ○ 2  |  Steps: 2/5 complete          │
└──────────────────────────────────────────────────┘
```

**Status indicators** (left sidebar, per step):
- `○` gray circle — pending (not started)
- `◉` blue circle with pulse animation — running
- `✓` green check — done
- `✗` red X — error

### 7. Error Handling

| Scenario | Behavior |
|---|---|
| Run not found / invalid runID | 404 page |
| Run already done/failed | Render completed view, no SSE connect |
| SSE connection lost mid-run | Browser auto-reconnects (native EventSource behavior); if run completes during disconnect, `onerror` sees `runStatus=done` and closes gracefully |
| Redis unavailable | SSE handler returns 503; page shows "Live tracking unavailable" |
| Callback nil (tests, no Redis configured) | Engine skips silently, no events published |
| Empty steps (0-step pipeline) | `OnRunStart` → `OnRunComplete` immediately |
| Browser tab hidden/background | `EventSource` keeps connection; no events lost |

### 8. Routes

| Method | Path | Handler | Auth |
|---|---|---|---|
| GET | `/pipelines/:name/runs/:runID/live` | `pipelineRunLivePage` | Cookie |
| GET | `/pipelines/:name/runs/:runID/live/watch` | `watchPipelineRunLive` | Cookie |

Run list page (`/pipelines/:name/runs`) adds a "Live" link on rows where `status = running`.

### 9. Files Summary

| File | Change |
|---|---|
| `pkg/pipeline/progress.go` | **New**: `StepCallback` interface, `StepProgressEvent` struct |
| `pkg/pipeline/engine.go` | Add `callback StepCallback` field; call hooks in `executePipeline`/`executeStep` |
| `internal/server/pipeline.go` | **New**: `pipelineStepCallback` (go-redis XAdd); fx provider factory |
| `internal/modules/web/pipeline_webservice.go` | Add `pipelineRunLivePage` + `watchPipelineRunLive` handlers |
| `pkg/views/pages/pipeline_run_live.templ` | **New**: Live dashboard page template |
| `public/js/pipeline-run-live.js` | **New**: Alpine.js `pipelineRunLive` component |
| `pkg/views/partials/pipeline_runs.templ` | Add "Live" link on running run rows |
| `internal/store/store.go` | Add `GetRunWithSteps()` if needed (existing methods may suffice) |
| `internal/modules/web/types.go` | Register new template render function |

### 10. Conventions

- All text in English
- No Watermill for stream publish/subscribe — raw go-redis `XAdd`/`XRead`
- Callback nil-safe — engine checks before calling
- StepIndex -1 convention for run-level events (start/complete/failed)
- Cookie auth via existing middleware, same origin SSE
- Stream TTL: 24h failsafe on `OnRunStart`, shortened to 5m on `OnRunComplete` for graceful SSE drain
- `data-testid` attributes on all interactive elements
- Store methods in `store.go` only, all queries via ent-generated client

### 11. Out of Scope

- Stream archival or replay beyond 5-minute TTL
- Concurrent run live view (multiple SSE clients watching same run) — initial version works for single viewer
- WebSocket transport — SSE is sufficient for unidirectional server→browser push
- Auto-reconnect with event ID replay (Last-Event-ID header) — first version uses simple reconnect + page refresh
- Step-level time elapsed while step is still running (only sent at completion)
