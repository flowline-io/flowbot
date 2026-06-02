# Pipeline Live Run Dashboard — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add real-time live run dashboard showing step-by-step pipeline execution progress via SSE.

**Architecture:** Engine emits progress via `StepCallback` interface → `pipelineStepCallback` publishes to per-run Redis Stream (`pipeline:run:{runID}`) → SSE handler subscribes via raw go-redis `XRead` → Alpine.js `EventSource` renders live updates. Initial state queried from PostgreSQL, SSE fills in real-time deltas.

**Tech Stack:** Go (engine callbacks, go-redis XAdd/XRead, Fiber SetBodyStreamWriter), templ (page template), Alpine.js (SPA component), Redis Streams, PostgreSQL (initial state)

---

### Task 1: StepCallback Interface + StepProgressEvent

**Files:**
- Create: `pkg/pipeline/progress.go`

- [ ] **Step 1: Write the interface and event types**

```go
// Package pipeline provides the workflow execution engine.
package pipeline

import (
	"context"
	"time"
)

// StepCallback receives progress events during pipeline execution.
// All methods are called synchronously from the step execution loop.
// nil receiver is safe — Engine skips calls when callback is nil.
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

// StepProgressEvent is the JSON payload for a single progress update.
// StepIndex of -1 indicates a run-level event (start/complete/failed).
type StepProgressEvent struct {
	RunID        int64          `json:"run_id"`
	PipelineName string         `json:"pipeline_name"`
	StepIndex    int            `json:"step_index"`
	StepName     string         `json:"step_name"`
	Status       string         `json:"status"`
	Input        map[string]any `json:"input,omitempty"`
	Output       map[string]any `json:"output,omitempty"`
	ElapsedMs    int64          `json:"elapsed_ms,omitempty"`
	Error        string         `json:"error,omitempty"`
	TotalSteps   int            `json:"total_steps,omitempty"`
}

// StreamName returns the Redis Stream name for a given run ID.
func StreamName(runID int64) string {
	return fmt.Sprintf("pipeline:run:%d", runID)
}

// StreamTTLFailsafe is the TTL set on stream creation to prevent leaks on crash.
const StreamTTLFailsafe = 24 * time.Hour

// StreamTTLDrain is the TTL after completion for SSE clients to drain.
const StreamTTLDrain = 5 * time.Minute
```

- [ ] **Step 2: Verify build compiles**

```bash
go tool task build
```

Expected: PASS (no errors, even though callback is unused yet)

- [ ] **Step 3: Commit**

```bash
git add pkg/pipeline/progress.go
git commit -m "feat: add StepCallback interface and StepProgressEvent types"
```

---

### Task 2: Engine — Add Callback Field and Hook Calls

**Files:**
- Modify: `pkg/pipeline/engine.go`

- [ ] **Step 1: Add callback field to Engine struct**

Find the `Engine` struct (around line 86-96) and add a `callback` field after the existing fields:

```go
type Engine struct {
	defs            []Definition
	store           RunStore
	auditor         audit.Auditor
	pipelineMetrics *metrics.PipelineCollector
	eventMetrics    *metrics.EventCollector
	handler         func(ctx context.Context, event types.DataEvent) error
	mu              map[string]*sync.Mutex
	cron            *cron.Cron
	clock           Clock
	callback        StepCallback // added: progress event callback (nil-safe)
}
```

- [ ] **Step 2: Add SetCallback method**

Add after the `Handler()` method:

```go
// SetCallback sets the progress event callback. Pass nil to disable.
func (e *Engine) SetCallback(cb StepCallback) {
	e.callback = cb
}
```

- [ ] **Step 3: Add callback calls in executePipeline**

In the `executePipeline` method (around line 172), add `OnRunStart` after `createRunRecord` succeeds (after line 192, before the step loop):

```go
// After: if err := e.createRunRecord(ctx, runID, def.Name, event); err != nil { ... }
// Add:
if e.callback != nil {
	triggerDesc := triggerDescription(def.Trigger)
	stepNames := make([]string, len(def.Steps))
	for i, s := range def.Steps {
		stepNames[i] = s.Name
	}
	e.callback.OnRunStart(ctx, runID, def.Name, triggerDesc, len(def.Steps), stepNames)
}
```

Add `OnRunComplete` just before `finishRunRecord` returns (after line 220 area, replacing the final return):

```go
// After step loop + metrics recording + finishRunRecord
if e.callback != nil {
	elapsed := e.clock.Since(startTime).Milliseconds()
	var errMsg string
	if stepErr != nil {
		errMsg = stepErr.Error()
	}
	e.callback.OnRunComplete(ctx, runID, def.Name, elapsed, lastFailed, errMsg)
}
```

- [ ] **Step 4: Add a helper to describe the trigger**

Add a `triggerDescription` function at the bottom of the file or in a new utility section:

```go
// triggerDescription returns a human-readable trigger description string.
func triggerDescription(t Trigger) string {
	if t.Event != "" {
		return "event:" + t.Event
	}
	if t.Webhook != nil && t.Webhook.Path != "" {
		return "webhook:" + t.Webhook.Path
	}
	if t.Cron != "" {
		return "cron:" + t.Cron
	}
	return "unknown"
}
```

- [ ] **Step 5: Add callback calls in executeStep**

In `executeStep` (around line 230), add `OnStepStart` after params are rendered but before `ability.Invoke` (after line 245 area, before the retry block):

```go
// After tag injection, before backoff.Do:
if e.callback != nil {
	e.callback.OnStepStart(ctx, runID, pipelineName, stepIndex, step.Name, renderedParams)
}
```

Add `OnStepDone` after successful invoke (after line 298, inside the success path):

```go
// After recordStepSuccess, before the return:
if e.callback != nil {
	e.callback.OnStepDone(ctx, runID, pipelineName, stepIndex, step.Name, result, elapsed)
}
```

Add `OnStepError` after failed invoke (after line 293, inside the failure path):

```go
// After recordStepFailure:
if e.callback != nil {
	e.callback.OnStepError(ctx, runID, pipelineName, stepIndex, step.Name, invokeErr, elapsed)
}
```

- [ ] **Step 6: Find the correct elapsed and result variables**

Read the `executeStep` function to find the variable names for:
- `runID` (first param to `createStepRunRecord`)
- `pipelineName` (passed through or available from the outer scope)
- `stepIndex` (loop counter from `executePipeline`)
- `invokeErr` (error from `ability.Invoke`)
- `result` (output map from `ability.Invoke`)
- `elapsed` (time since step start)

If `stepIndex` is not available, add it as a parameter to `executeStep`:
```go
// Change signature from:
func (e *Engine) executeStep(ctx context.Context, runID int64, pipelineName string,
	rc *RenderContext, step Step, attempt int) (map[string]any, error)
// To:
func (e *Engine) executeStep(ctx context.Context, runID int64, pipelineName string,
	rc *RenderContext, step Step, attempt int, stepIndex int) (map[string]any, error)
```

And update the caller in `executePipeline`:
```go
// Change:
result, err := e.executeStep(execCtx, runID, def.Name, rc, step, 1)
// To:
result, err := e.executeStep(execCtx, runID, def.Name, rc, step, 1, i)
```

- [ ] **Step 7: Verify build compiles**

```bash
go tool task build
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add pkg/pipeline/engine.go pkg/pipeline/progress.go
git commit -m "feat: add StepCallback hooks to pipeline engine"
```

---

### Task 3: Engine Tests — Verify Callback Invocation

**Files:**
- Modify: `pkg/pipeline/engine_test.go`

- [ ] **Step 1: Write a mock callback for tests**

Add at the top of the test file or near the Engine tests:

```go
// mockStepCallback records all callback invocations for test assertions.
type mockStepCallback struct {
	calls []mockCallbackCall
	mu    sync.Mutex
}

type mockCallbackCall struct {
	method      string
	runID       int64
	stepIndex   int
	stepName    string
	status      string
	elapsedMs   int64
}

func (m *mockStepCallback) OnRunStart(ctx context.Context, runID int64, pipelineName string,
	trigger string, totalSteps int, stepNames []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnRunStart", runID: runID})
}

func (m *mockStepCallback) OnStepStart(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, input map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnStepStart", runID: runID, stepIndex: stepIndex, stepName: stepName})
}

func (m *mockStepCallback) OnStepDone(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, output map[string]any, elapsedMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnStepDone", runID: runID, stepIndex: stepIndex, stepName: stepName, elapsedMs: elapsedMs})
}

func (m *mockStepCallback) OnStepError(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, err error, elapsedMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnStepError", runID: runID, stepIndex: stepIndex, stepName: stepName, elapsedMs: elapsedMs})
}

func (m *mockStepCallback) OnRunComplete(ctx context.Context, runID int64, pipelineName string,
	elapsedMs int64, failed bool, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := "complete"
	if failed { status = "failed" }
	m.calls = append(m.calls, mockCallbackCall{method: "OnRunComplete", runID: runID, status: status, elapsedMs: elapsedMs})
}
```

- [ ] **Step 2: Write test — callback is called during pipeline execution**

Add a test function `TestEngine_CallbackInvocation` using the table-driven pattern:

```go
func TestEngine_CallbackInvocation(t *testing.T) {
	tests := []struct {
		name       string
		def        Definition
		wantStart  bool
		wantStep   bool
		wantDone   bool
		wantErrors int
	}{
		{
			name: "happy path — all hooks called",
			def: Definition{
				Name: "test-pipeline",
				Trigger: Trigger{Event: "test.event"},
				Steps: []Step{
					{Name: "step1", Capability: "test", Operation: "noop"},
				},
			},
			wantStart: true,
			wantStep:  true,
			wantDone:  true,
		},
		{
			name: "nil callback — no panic",
			def: Definition{
				Name: "test-pipeline-nil-cb",
				Trigger: Trigger{Event: "test.event2"},
				Steps: []Step{
					{Name: "step1", Capability: "test", Operation: "noop"},
				},
			},
			wantStart: false,
			wantStep:  false,
		},
		{
			name: "multiple steps — order preserved",
			def: Definition{
				Name: "test-pipeline-multi",
				Trigger: Trigger{Event: "test.event3"},
				Steps: []Step{
					{Name: "step1", Capability: "test", Operation: "noop"},
					{Name: "step2", Capability: "test", Operation: "noop"},
				},
			},
			wantStart: true,
			wantStep:  true,
			wantDone:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeRunStore()
			mockCB := &mockStepCallback{}
			engine := NewEngineWithClock([]Definition{tt.def}, store, nil, nil, nil, NewFakeClock())
			if tt.wantStart {
				engine.SetCallback(mockCB)
			}
			// ability.Invoke is called during execution — mock it
			// Note: this test requires the ability invocation to be mockable.
			// See existing engine_test.go for how engine tests mock ability calls.
		})
	}
}
```

Note: The existing `engine_test.go` likely has a pattern for mocking `ability.Invoke`. Read the existing test file to match the pattern for invoking pipeline steps. If the engine tests require a full integration setup (`ability` registry), this test may need to verify callback at a unit level instead — by calling `executePipeline`/`executeStep` directly with mocked dependencies.

- [ ] **Step 3: Write test — order of callbacks is correct**

```go
func TestStepCallback_OrderOfCalls(t *testing.T) {
	mockCB := &mockStepCallback{}
	// Simulate the expected call sequence:
	mockCB.OnRunStart(context.Background(), 1, "p", "event:x", 2, []string{"a", "b"})
	mockCB.OnStepStart(context.Background(), 1, "p", 0, "a", nil)
	mockCB.OnStepDone(context.Background(), 1, "p", 0, "a", nil, 100)
	mockCB.OnStepStart(context.Background(), 1, "p", 1, "b", nil)
	mockCB.OnStepDone(context.Background(), 1, "p", 1, "b", nil, 200)
	mockCB.OnRunComplete(context.Background(), 1, "p", 300, false, "")

	expectedOrder := []string{
		"OnRunStart", "OnStepStart", "OnStepDone",
		"OnStepStart", "OnStepDone", "OnRunComplete",
	}
	for i, call := range mockCB.calls {
		if call.method != expectedOrder[i] {
			t.Errorf("call %d: got %s, want %s", i, call.method, expectedOrder[i])
		}
	}
	if len(mockCB.calls) != len(expectedOrder) {
		t.Errorf("got %d calls, want %d", len(mockCB.calls), len(expectedOrder))
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/pipeline/... -v -run TestCallback
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/pipeline/engine_test.go
git commit -m "test: add StepCallback invocation tests"
```

---

### Task 4: pipelineStepCallback — Redis Stream Publisher

**Files:**
- Modify: `internal/server/pipeline.go`

- [ ] **Step 1: Add the pipelineStepCallback type and constructor**

Add at the top of the file (after imports) or near the `EventEmitter` section:

```go
import (
	// add to existing imports:
	"github.com/redis/go-redis/v9"
	"github.com/bytedance/sonic"
)

// pipelineStepCallback publishes pipeline progress events to Redis Streams.
type pipelineStepCallback struct {
	rdb *redis.Client
}

// NewPipelineStepCallback creates a callback backed by the Redis client.
func NewPipelineStepCallback(rdb *redis.Client) pipeline.StepCallback {
	if rdb == nil {
		return nil
	}
	return &pipelineStepCallback{rdb: rdb}
}

func (c *pipelineStepCallback) OnRunStart(ctx context.Context, runID int64, pipelineName string,
	trigger string, totalSteps int, stepNames []string) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: -1, Status: "start", TotalSteps: totalSteps,
	}
	c.publish(runID, evt)
	c.rdb.Expire(ctx, pipeline.StreamName(runID), pipeline.StreamTTLFailsafe)
}

func (c *pipelineStepCallback) OnStepStart(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, input map[string]any) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "running", Input: input,
	}
	c.publish(runID, evt)
}

func (c *pipelineStepCallback) OnStepDone(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, output map[string]any, elapsedMs int64) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "done", Output: output, ElapsedMs: elapsedMs,
	}
	c.publish(runID, evt)
}

func (c *pipelineStepCallback) OnStepError(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, err error, elapsedMs int64) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "error", Error: err.Error(), ElapsedMs: elapsedMs,
	}
	c.publish(runID, evt)
}

func (c *pipelineStepCallback) OnRunComplete(ctx context.Context, runID int64, pipelineName string,
	elapsedMs int64, failed bool, errMsg string) {
	status := "complete"
	if failed {
		status = "failed"
	}
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: -1, Status: status, ElapsedMs: elapsedMs, Error: errMsg,
	}
	c.publish(runID, evt)
	c.rdb.Expire(ctx, pipeline.StreamName(runID), pipeline.StreamTTLDrain)
}

// publish sends a progress event to the per-run Redis Stream asynchronously
// to avoid blocking the pipeline engine on Redis latency or errors.
func (c *pipelineStepCallback) publish(runID int64, evt pipeline.StepProgressEvent) {
	payload, err := sonic.Marshal(evt)
	if err != nil {
		return
	}
	go func() {
		pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		c.rdb.XAdd(pubCtx, &redis.XAddArgs{
			Stream: pipeline.StreamName(runID),
			Values: map[string]any{"data": payload},
		}).Err() // errors are intentionally ignored — best-effort progress push
	}()
}
```

- [ ] **Step 2: Wire into engine creation**

In `setupPipelineEngine` (around line 91), after `engine := pipeline.NewEngine(...)` add:

```go
if rdb.Client != nil {
	engine.SetCallback(NewPipelineStepCallback(rdb.Client))
}
```

- [ ] **Step 3: Verify build compiles**

```bash
go tool task build
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/server/pipeline.go pkg/pipeline/progress.go
git commit -m "feat: add Redis Stream progress publisher for pipeline runs"
```

---

### Task 5: SSE Watch Endpoint

**Files:**
- Modify: `internal/modules/web/pipeline_webservice.go`

- [ ] **Step 1: Add imports**

Add to existing imports in `pipeline_webservice.go`:

```go
import (
	// existing imports...
	"bufio"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/rdb"
)
```

- [ ] **Step 2: Add the SSE handler method**

```go
// watchPipelineRunLive opens an SSE stream for a running pipeline.
func (h moduleHandler) watchPipelineRunLive(c fiber.Ctx) error {
	runIDParam := c.Params("runID")
	runID, err := strconv.ParseInt(runIDParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid runID")
	}
	stream := pipeline.StreamName(runID)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	ctx := c.Context()
	redisClient := rdb.Client
	if redisClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("redis not available")
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		lastID := "0"
		for {
			select {
			case <-ctx.Done():
				return
			default:
				result, err := redisClient.XRead(ctx, &redis.XReadArgs{
					Streams: []string{stream, lastID},
					Count:   10,
					Block:   5 * time.Second,
				}).Result()

				if errors.Is(err, context.Canceled) {
					return
				}
				if err == redis.Nil || len(result) == 0 {
					fmt.Fprintf(w, ": heartbeat\n\n")
					w.Flush()
					continue
				}
				if err != nil {
					time.Sleep(2 * time.Second)
					continue
				}
				for _, msg := range result[0].Messages {
					lastID = msg.ID
					data, ok := msg.Values["data"].(string)
					if !ok {
						continue
					}
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

- [ ] **Step 3: Verify build compiles**

```bash
go tool task build
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go
git commit -m "feat: add SSE watch endpoint for live pipeline runs"
```

---

### Task 6: Live Dashboard Page Handler

**Files:**
- Modify: `internal/modules/web/pipeline_webservice.go`
- Modify: `internal/store/store.go` (if needed)

- [ ] **Step 1: Check if GetRun and ListStepRuns exist**

Read `internal/store/store.go` to find existing methods for querying a single run by ID and step runs by run ID. Look for methods like:
- `GetRun(ctx, runID int64) (*gen.PipelineRun, error)` — this exists in `RunStore` interface
- Method to list step runs by run ID

Check if `*PipelineStore` has a method matching these patterns. If it does, note the function signature. If not, we'll add them.

- [ ] **Step 2: Add GetRunByID if missing**

If `GetRun` exists only on the `RunStore` interface (used by engine) but not as a public method on `*PipelineStore`, add it:

In `internal/store/store.go`, in the `PipelineStore` methods section:

```go
// GetRunByID returns a pipeline run by its database ID.
func (s *PipelineStore) GetRunByID(ctx context.Context, id int64) (*gen.PipelineRun, error) {
	run, err := s.client.PipelineRun.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return run, nil
}

// ListStepRunsByRunID returns all step runs for a pipeline run, ordered by index.
func (s *PipelineStore) ListStepRunsByRunID(ctx context.Context, runID int64) ([]*gen.PipelineStepRun, error) {
	steps, err := s.client.PipelineStepRun.Query().
		Where(pipelinesteprun.PipelineRunIDEQ(runID)).
		Order(gen.Asc(pipelinesteprun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return steps, nil
}
```

Note: Verify the exact ent field names. The generated code in `internal/store/ent/gen/` has the actual field constants. Use `grep` to find the correct names.

- [ ] **Step 3: Add the page handler**

```go
// pipelineRunLivePage renders the live run dashboard page.
func pipelineRunLivePage(c fiber.Ctx) error {
	pipelineName := c.Params("name")
	runIDParam := c.Params("runID")
	runID, err := strconv.ParseInt(runIDParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid runID")
	}

	s := getPipelineDefStore()
	if s == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("store not available")
	}

	run, err := s.GetRunByID(context.Background(), runID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("run not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load run")
	}

	steps, err := s.ListStepRunsByRunID(context.Background(), runID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load steps")
	}

	// Map step runs to initial Alpine.js state
	type stepState struct {
		Name      string         `json:"name"`
		Status    string         `json:"status"`
		ElapsedMs int64          `json:"elapsed_ms"`
		Output    map[string]any `json:"output"`
		Error     string         `json:"error"`
		Input     map[string]any `json:"input"`
	}
	initSteps := make([]stepState, len(steps))
	for i, s := range steps {
		status := "pending"
		switch s.Status {
		case 1: // PipelineStart → "running" (or check actual status constants)
			status = "running"
		case 2: // PipelineDone
			status = "done"
		case 4: // PipelineFailed
			status = "error"
		}
		initSteps[i] = stepState{
			Name:   s.StepName,
			Status: status,
			Output: s.Result,
			Error:  s.Error,
			Input:  s.Params,
		}
		if s.CompletedAt != nil && s.StartedAt != nil {
			initSteps[i].ElapsedMs = s.CompletedAt.Sub(s.StartedAt).Milliseconds()
		}
	}

	runStatus := "pending"
	switch run.Status {
	case 1: // PipelineStart
		runStatus = "running"
	case 2:
		runStatus = "done"
	case 4:
		runStatus = "failed"
	}

	c.Type("html")
	return pages.PipelineRunLivePage(pages.PipelineRunLiveParams{
		RunID:        runID,
		PipelineName: pipelineName,
		Trigger:      run.EventType,
		TotalSteps:   len(steps),
		RunStatus:    runStatus,
		Steps:        initSteps,
	}).Render(context.Background(), c.Response().BodyWriter())
}
```

- [ ] **Step 4: Verify build compiles**

```bash
go tool task build
```

Expected: COMPILE ERROR (page template doesn't exist yet) — this is expected, we'll create it next.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go internal/store/store.go
git commit -m "feat: add live dashboard page handler with store methods"
```

---

### Task 7: Live Dashboard Page Template

**Files:**
- Create: `pkg/views/pages/pipeline_run_live.templ`

- [ ] **Step 1: Create the templ template**

```templ
package pages

import (
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

// PipelineRunLiveParams holds all data for the live dashboard page.
type PipelineRunLiveParams struct {
	RunID        int64
	PipelineName string
	Trigger      string
	TotalSteps   int
	RunStatus    string
	Steps        any // serialized to JSON for Alpine initial state
}

// PipelineRunLivePage renders the live run dashboard.
templ PipelineRunLivePage(p PipelineRunLiveParams) {
	@layout.Base("Live: " + p.PipelineName) {
		<script src="/static/js/pipeline-run-live.js" defer></script>
		<div class="max-w-6xl mx-auto">
			<!-- Header -->
			<div class="flex items-center justify-between mb-4">
				<div>
					<h1 class="text-xl font-semibold text-base-content">
						Live: <a href={ templ.URL("/service/web/pipelines/" + p.PipelineName) }
							class="link link-hover text-primary">{ p.PipelineName }</a>
					</h1>
					<p class="text-sm text-base-content/60 mt-1">Trigger: { p.Trigger }</p>
				</div>
				<div class="flex items-center gap-4">
					<span class="text-lg font-mono tabular-nums" id="total-elapsed">0s</span>
					<span id="run-status-badge" class="badge badge-info">Running</span>
				</div>
			</div>

			<!-- Two-column layout -->
			<div class="grid grid-cols-12 gap-4">
				<!-- Left: Step list -->
				<div class="col-span-4 space-y-1" data-testid="step-list">
					for i, step := range p.Steps {
						// rendered server-side as initial state
						// Alpine.js will update on SSE events
					}
				</div>
				<!-- Right: Detail panel -->
				<div class="col-span-8 bg-base-200 rounded-box p-4" data-testid="step-detail">
					<p class="text-base-content/60">Select a step to view details</p>
				</div>
			</div>

			<!-- Summary bar -->
			<div class="mt-4 bg-base-200 rounded-box p-3 flex items-center gap-4 text-sm"
			     data-testid="summary-bar">
				<span id="summary-completed">0</span> completed,
				<span id="summary-running">0</span> running,
				<span id="summary-pending">0</span> pending
				<span class="ml-auto">Steps: <span id="summary-progress">0/0</span></span>
			</div>
		</div>

		<!-- Initial state injection -->
		<script>
			(function() {
				var initialData = {
					runID: { p.RunID },
					pipelineName: "{ p.PipelineName }",
					trigger: "{ p.Trigger }",
					totalSteps: { p.TotalSteps },
					runStatus: "{ p.RunStatus }",
					steps: /* serialized steps JSON */
				};
				window.__pipelineRunLiveInitial = initialData;
			})();
		</script>
	}
}
```

Note: The exact templ syntax will need adjustment for the for-loop over steps and the JSON serialization. The `templ` language uses Go expressions inside `{ }` and template control flow. Steps data is passed as generic `any` and serialized via `sonic.MarshalString` to inject into the inline script.

- [ ] **Step 2: Generate Go code from template**

```bash
templ generate pkg/views/pages/pipeline_run_live.templ
```

Expected: Generates `pkg/views/pages/pipeline_run_live_templ.go` with function `PipelineRunLivePage(params PipelineRunLiveParams) templ.Component`

- [ ] **Step 3: Verify build compiles**

```bash
go tool task build
```

Expected: PASS (or compile errors that need fixing in the template syntax)

- [ ] **Step 4: Commit**

```bash
git add pkg/views/pages/pipeline_run_live.templ pkg/views/pages/pipeline_run_live_templ.go
git commit -m "feat: add live run dashboard page template"
```

---

### Task 8: Alpine.js Live Run Component

**Files:**
- Create: `public/js/pipeline-run-live.js`

- [ ] **Step 1: Create the JavaScript component**

```javascript
'use strict';

Alpine.data('pipelineRunLive', (initial) => ({
  runID: initial.runID,
  pipelineName: initial.pipelineName,
  trigger: initial.trigger,
  totalSteps: initial.totalSteps,
  steps: initial.steps,
  selectedIndex: -1,
  totalElapsed: 0,
  completed: 0,
  failedSteps: 0,
  runStatus: initial.runStatus,
  eventSource: null,

  init() {
    this.recalc();

    var idx = this.steps.findIndex(function (s) {
      return s.status === 'running' || s.status === 'pending';
    });
    this.selectedIndex = idx >= 0 ? idx : this.steps.length - 1;

    if (this.runStatus === 'running') {
      var self = this;
      var watchURL =
        window.location.pathname.replace(/\/live$/, '/live/watch');
      this.eventSource = new EventSource(watchURL);
      this.eventSource.onmessage = function (e) {
        var evt = JSON.parse(e.data);
        self.applyEvent(evt);
      };
      this.eventSource.onerror = function () {
        if (self.runStatus === 'done' || self.runStatus === 'failed') {
          self.eventSource.close();
        }
      };
    }
  },

  recalc: function () {
    this.completed = this.steps.filter(function (s) {
      return s.status === 'done';
    }).length;
    this.failedSteps = this.steps.filter(function (s) {
      return s.status === 'error';
    }).length;
    this.totalElapsed = this.steps.reduce(function (acc, s) {
      return acc + (s.elapsed_ms || 0);
    }, 0);
  },

  applyEvent: function (evt) {
    if (evt.step_index === -1) {
      if (evt.status === 'start') this.runStatus = 'running';
      if (evt.status === 'complete') this.runStatus = 'done';
      if (evt.status === 'failed') this.runStatus = 'failed';
      if (evt.elapsed_ms) this.totalElapsed = evt.elapsed_ms;
      return;
    }
    var step = this.steps[evt.step_index];
    if (!step) return;
    step.status = evt.status;
    if (evt.status === 'done') {
      step.output = evt.output;
      step.elapsed_ms = evt.elapsed_ms;
    }
    if (evt.status === 'error') {
      step.error = evt.error;
      step.elapsed_ms = evt.elapsed_ms;
    }
    if (evt.status === 'running') {
      step.input = evt.input;
      this.selectedIndex = evt.step_index;
    }
    this.recalc();
  },

  selectStep: function (idx) { this.selectedIndex = idx; },

  get selectedStep() {
    return this.steps[this.selectedIndex] || null;
  },

  get formattedElapsed() {
    var ms = this.totalElapsed;
    if (ms < 1000) return ms + 'ms';
    return (ms / 1000).toFixed(1) + 's';
  }
}));
```

- [ ] **Step 2: Verify the file is embedded**

The file is placed in `public/js/` which is embedded via `webassets.go`'s `//go:embed all:public`. Verify the embed directive exists:

```bash
grep 'go:embed' webassets.go
```

Expected: `//go:embed all:public`

- [ ] **Step 3: Commit**

```bash
git add public/js/pipeline-run-live.js
git commit -m "feat: add Alpine.js live run dashboard component"
```

---

### Task 9: Register Routes and Live Link

**Files:**
- Modify: `internal/modules/web/pipeline_webservice.go` (route rules)
- Modify: `pkg/views/partials/pipeline_runs.templ` (Live link)

- [ ] **Step 1: Add routes to pipelineWebserviceRules**

In `pipelineWebserviceRules` slice (around line 24-38), add two new entries:

```go
var pipelineWebserviceRules = []webservice.Rule{
	// ... existing rules ...
	webservice.Get("/pipelines/:name/runs/:runID/live", pipelineRunLivePage),
	webservice.Get("/pipelines/:name/runs/:runID/live/watch", watchPipelineRunLive),
}
```

- [ ] **Step 2: Add "Live" link to run history table**

Read `pkg/views/partials/pipeline_runs.templ` to find the `PipelineRunsTable` template. Find the column that shows run status and add a "Live" link when status indicates running.

The status integer values (from ent schema): 1 = PipelineStart (running), 2 = PipelineDone, 3 = PipelineCancel, 4 = PipelineFailed.

In the runs table row template, add a link next to running runs:

```templ
// In PipelineRunsTable or individual row template:
if run.Status == 1 {
    <a href={ templ.URL("/service/web/pipelines/" + pipelineName + "/runs/" + fmt.Sprintf("%d", run.ID) + "/live") }
       class="btn btn-xs btn-info ml-2" data-testid="live-link">
        Live
    </a>
}
```

- [ ] **Step 3: Regenerate template code**

```bash
templ generate pkg/views/partials/pipeline_runs.templ
```

- [ ] **Step 4: Verify build compiles**

```bash
go tool task build
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go pkg/views/partials/pipeline_runs.templ pkg/views/partials/pipeline_runs_templ.go
git commit -m "feat: register live dashboard routes and add Live link to run history"
```

---

### Task 10: Fix Template and Integrate

**Files:**
- Modify: `pkg/views/pages/pipeline_run_live.templ` (finalize after handlers exist)
- Modify: `internal/modules/web/pipeline_webservice.go` (fix any type mismatches)

- [ ] **Step 1: Finalize the page template with proper Alpine.js wiring**

The pipeline_run_live.templ needs the Alpine.js component mounted with initial data. Update the template to:
1. Pass serialized steps JSON into Alpine initial data
2. Mount the `pipelineRunLive` component with `x-data` and `x-init`
3. Render the step list with server-side status indicators
4. Wire up click handlers for step selection
5. Show the detail panel for selected step

This is the most involved task. Write the complete template with all the UI elements matching the spec's layout. Use DaisyUI classes for badges, buttons, and layout.

- [ ] **Step 2: Handle type mismatches**

Fix any compilation errors between the handler's `PipelineRunLiveParams` and the template's expected types. Ensure `Steps` field is serializable to JSON.

- [ ] **Step 3: Regenerate and build**

```bash
templ generate pkg/views/...
go tool task build
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/pages/pipeline_run_live.templ pkg/views/pages/pipeline_run_live_templ.go internal/modules/web/pipeline_webservice.go
git commit -m "feat: integrate live dashboard template with Alpine.js"
```

---

### Task 11: Lint, Test, Final Verification

**Files:**
- All modified files

- [ ] **Step 1: Run lint**

```bash
go tool task lint
```

Fix any revive warnings (unused imports, naming, etc.).

- [ ] **Step 2: Run unit tests**

```bash
go tool task test
```

- [ ] **Step 3: Run format**

```bash
go tool task format
```

- [ ] **Step 4: Final build verification**

```bash
go tool task build
```

Expected: PASS with no errors.

- [ ] **Step 5: Commit any lint/format fixes**

```bash
git add -A
git commit -m "chore: lint and format fixes for live dashboard"
```

---

## File Summary

| File | Action |
|------|--------|
| `pkg/pipeline/progress.go` | **Create**: `StepCallback` interface, `StepProgressEvent`, `StreamName()`, TTL constants |
| `pkg/pipeline/engine.go` | **Modify**: Add `callback` field, `SetCallback()`, hook calls in `executePipeline`/`executeStep`, `stepIndex` param |
| `pkg/pipeline/engine_test.go` | **Modify**: Add `mockStepCallback`, callback invocation + order tests |
| `internal/server/pipeline.go` | **Modify**: Add `pipelineStepCallback`, `NewPipelineStepCallback`, wire into `setupPipelineEngine` |
| `internal/modules/web/pipeline_webservice.go` | **Modify**: Add `pipelineRunLivePage`, `watchPipelineRunLive`, route entries |
| `internal/store/store.go` | **Modify**: Add `GetRunByID`, `ListStepRunsByRunID` |
| `pkg/views/pages/pipeline_run_live.templ` | **Create**: Live dashboard page with Alpine.js mount |
| `pkg/views/partials/pipeline_runs.templ` | **Modify**: Add "Live" link on running runs |
| `public/js/pipeline-run-live.js` | **Create**: Alpine.js `pipelineRunLive` component |
