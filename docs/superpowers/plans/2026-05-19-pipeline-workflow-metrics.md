# Pipeline / Workflow Metrics Monitoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Prometheus-based business metrics for pipeline execution, workflow execution, event processing, and ability invocation with full-dimensional labeling.

**Architecture:** New `pkg/metrics/` package with four typed collectors (Pipeline, Workflow, Event, Ability) that bridge to `pkg/stats/` Pushgateway infrastructure. `pkg/stats/` gains `*Stats` struct + vec registration methods. Collectors inject into engines/registry via constructor parameters or global setters.

**Tech Stack:** Go 1.26+, Prometheus client_golang, testutil, testify, Fx

---

### Task 1: Add vector metric support to `pkg/stats/`

**Files:**
- Modify: `pkg/stats/stats.go`
- Modify: `pkg/stats/stats_test.go`

pkg/stats currently only supports flat Counter/Gauge with fixed ConstLabels. We need `Stats` struct wrapping the global registry plus `RegisterCounterVec`, `RegisterGaugeVec`, `RegisterHistogramVec` methods. Also add `IsInitialized()` so collectors know when to return no-op. When `Init()` is called (gated by `config.App.Metrics.Enabled`), it sets `initialized = true`. When metrics is disabled, `Init()` is never called, `NewStats()` returns nil, all collectors become no-op.

- [ ] **Step 1: Write the failing test**

In `pkg/stats/stats_test.go`, after the existing tests, append:

```go
func TestRegisterVecMetrics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T, s *Stats)
	}{
		{
			name: "counter vec registers and works",
			fn: func(t *testing.T, s *Stats) {
				cv := s.RegisterCounterVec("test_counter_vec_total", "help", "label_a")
				require.NotNil(t, cv)
				cv.WithLabelValues("val1").Inc()
				cv.WithLabelValues("val1").Inc()
				cv.WithLabelValues("val2").Inc()
			},
		},
		{
			name: "gauge vec registers and works",
			fn: func(t *testing.T, s *Stats) {
				gv := s.RegisterGaugeVec("test_gauge_vec", "help", "label_a")
				require.NotNil(t, gv)
				gv.WithLabelValues("val1").Set(42)
				gv.WithLabelValues("val2").Inc()
				gv.WithLabelValues("val2").Dec()
			},
		},
		{
			name: "histogram vec registers and works",
			fn: func(t *testing.T, s *Stats) {
				hv := s.RegisterHistogramVec("test_histogram_vec_seconds", "help", "label_a")
				require.NotNil(t, hv)
				hv.WithLabelValues("val1").Observe(1.5)
				hv.WithLabelValues("val2").Observe(0.5)
			},
		},
		{
			name: "duplicate registration returns same vec",
			fn: func(t *testing.T, s *Stats) {
				cv1 := s.RegisterCounterVec("test_dup_vec_total", "help", "l")
				cv2 := s.RegisterCounterVec("test_dup_vec_total", "help", "l")
				assert.Same(t, cv1, cv2)
			},
		},
		{
			name: "nil stats panics on register",
			fn: func(t *testing.T, s *Stats) {
				assert.Panics(t, func() {
					var nilStats *Stats
					nilStats.RegisterCounterVec("x", "h", "l")
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewStats()
			tt.fn(t, s)
		})
	}
}

func TestNewStatsWhenNotInitialized(t *testing.T) {
	t.Run("returns nil when Init not called", func(t *testing.T) {
		old := initialized
		initialized = false
		defer func() { initialized = old }()
		assert.Nil(t, NewStats())
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/stats/ -run "TestRegisterVecMetrics|TestNewStats" -v
```
Expected: compilation error (`Stats` undefined, `NewStats` undefined).

- [ ] **Step 3: Add initialized flag to stats.go**

In `pkg/stats/stats.go`, after the global var block (after line 69), add:

```go
var initialized bool
```

In the `Init` function, in `once.Do(...)`, add as the first line:

```go
	once.Do(func() {
		initialized = true
		// ... existing config handling ...
	})
```

- [ ] **Step 4: Add `Stats` struct and `NewStats`**

After the `initialized` var and before `MetricsConfig`:

```go
// Stats provides access to the global Prometheus registry for creating vector metrics.
type Stats struct {
	vecCounters map[string]*prometheus.CounterVec
	vecGauges   map[string]*prometheus.GaugeVec
	vecHistos   map[string]*prometheus.HistogramVec
}

// NewStats creates a Stats wrapper around the global Prometheus registry.
// Returns nil when metrics has not been initialized (metrics.enabled=false).
func NewStats() *Stats {
	if !initialized {
		return nil
	}
	return &Stats{
		vecCounters: make(map[string]*prometheus.CounterVec),
		vecGauges:   make(map[string]*prometheus.GaugeVec),
		vecHistos:   make(map[string]*prometheus.HistogramVec),
	}
}
```

- [ ] **Step 5: Add `RegisterCounterVec` method**

```go
// RegisterCounterVec creates or returns an existing CounterVec registered with the global registry.
func (s *Stats) RegisterCounterVec(name, help string, labelNames ...string) *prometheus.CounterVec {
	if s == nil {
		panic("stats: RegisterCounterVec called on nil Stats")
	}
	mu.Lock()
	defer mu.Unlock()

	if cv, exists := s.vecCounters[name]; exists {
		return cv
	}

	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labelNames)
	registry.MustRegister(cv)
	s.vecCounters[name] = cv
	return cv
}
```

- [ ] **Step 6: Add `RegisterGaugeVec` method**

```go
// RegisterGaugeVec creates or returns an existing GaugeVec registered with the global registry.
func (s *Stats) RegisterGaugeVec(name, help string, labelNames ...string) *prometheus.GaugeVec {
	if s == nil {
		panic("stats: RegisterGaugeVec called on nil Stats")
	}
	mu.Lock()
	defer mu.Unlock()

	if gv, exists := s.vecGauges[name]; exists {
		return gv
	}

	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labelNames)
	registry.MustRegister(gv)
	s.vecGauges[name] = gv
	return gv
}
```

- [ ] **Step 7: Add `RegisterHistogramVec` method**

```go
// RegisterHistogramVec creates or returns an existing HistogramVec registered with the global registry.
func (s *Stats) RegisterHistogramVec(name, help string, labelNames ...string) *prometheus.HistogramVec {
	if s == nil {
		panic("stats: RegisterHistogramVec called on nil Stats")
	}
	mu.Lock()
	defer mu.Unlock()

	if hv, exists := s.vecHistos[name]; exists {
		return hv
	}

	hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.DefBuckets,
	}, labelNames)
	registry.MustRegister(hv)
	s.vecHistos[name] = hv
	return hv
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
go test ./pkg/stats/ -run "TestRegisterVecMetrics|TestNewStats" -v
```
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add pkg/stats/stats.go pkg/stats/stats_test.go
git commit -m "feat(stats): add Stats struct with RegisterCounterVec, RegisterGaugeVec, RegisterHistogramVec"
```

---

### Task 2: Create shared helpers in `pkg/metrics/types.go`

**Files:**
- Create: `pkg/metrics/types.go`
- Create: `pkg/metrics/types_test.go`

- [ ] **Step 1: Create types.go**

```go
package metrics

import (
	"log"
	"regexp"
	"strings"
)

var safeLabelRe = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

func sanitizeLabel(v string) string {
	if len(v) > 128 {
		v = v[:128]
	}
	return safeLabelRe.ReplaceAllString(v, "_")
}

func recoverLog(metricName string) {
	if r := recover(); r != nil {
		log.Printf("[metrics] %s panic: %v", metricName, r)
	}
}
```

- [ ] **Step 2: Create types_test.go**

```go
package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no change", input: "archive-items", want: "archive-items"},
		{name: "spaces to underscore", input: "my pipeline", want: "my_pipeline"},
		{name: "special chars", input: "a/b?c:d#e", want: "a_b_c_d_e"},
		{name: "truncate long values", input: strings.Repeat("x", 200), want: strings.Repeat("x", 128)},
		{name: "empty string", input: "", want: ""},
		{name: "chinese characters", input: "管道", want: "___"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeLabel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./pkg/metrics/ -run TestSanitizeLabel -v
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/metrics/types.go pkg/metrics/types_test.go
git commit -m "feat(metrics): add sanitizeLabel and recoverLog helpers"
```

---

### Task 3: Create `PipelineCollector`

**Files:**
- Create: `pkg/metrics/pipeline.go`
- Create: `pkg/metrics/pipeline_test.go`

- [ ] **Step 1: Write failing test**

`pkg/metrics/pipeline_test.go`:
```go
package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewPipelineCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewPipelineCollector(nil)
		assert.NotNil(t, c)
		c.IncRunTotal("p", "done")
		c.ObserveRunDuration("p", "done", 1.5)
		c.IncStepTotal("p", "s", "done")
		c.ObserveStepDuration("p", "s", "cap", "done", 0.5)
		c.IncStepRetry("p", "s")
		c.IncResume("p")
	})
}

func TestPipelineCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewPipelineCollector(s)

	c.IncRunTotal("archive-items", "done")
	c.IncRunTotal("archive-items", "done")
	c.IncRunTotal("archive-items", "cancel")
	c.IncRunTotal("sync-bookmarks", "done")

	c.IncStepRetry("archive-items", "step1")
	c.IncStepRetry("archive-items", "step1")

	c.IncResume("archive-items")

	expected := `
# HELP pipeline_run_total Runs by pipeline and status
# TYPE pipeline_run_total counter
pipeline_run_total{pipeline="archive-items",status="done"} 2
pipeline_run_total{pipeline="archive-items",status="cancel"} 1
pipeline_run_total{pipeline="sync-bookmarks",status="done"} 1
`
	err := testutil.CollectAndCompare(c.runTotal, strings.NewReader(expected), "pipeline_run_total")
	assert.NoError(t, err)
}

func TestPipelineCollector_HistogramMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewPipelineCollector(s)

	c.ObserveRunDuration("p1", "done", 2.0)
	c.ObserveRunDuration("p1", "done", 3.0)
	c.ObserveStepDuration("p1", "step1", "bookmark", "done", 0.5)
	assert.NotNil(t, c.runDuration)
	assert.NotNil(t, c.stepDuration)
}

func TestPipelineCollector_LabelsSanitized(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewPipelineCollector(s)

	c.IncRunTotal("my pipeline!", "done")
	c.IncStepTotal("p", "step with spaces", "done")

	// Must not panic with special chars in labels
	assert.NotPanics(t, func() {
		c.IncRunTotal("name with / and ?", "done")
	})
}

func TestPipelineCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewPipelineCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncRunTotal", fn: func() { c.IncRunTotal("p", "done") }},
		{name: "ObserveRunDuration", fn: func() { c.ObserveRunDuration("p", "done", 1.0) }},
		{name: "IncStepTotal", fn: func() { c.IncStepTotal("p", "s", "done") }},
		{name: "ObserveStepDuration", fn: func() { c.ObserveStepDuration("p", "s", "c", "done", 1.0) }},
		{name: "IncStepRetry", fn: func() { c.IncStepRetry("p", "s") }},
		{name: "IncResume", fn: func() { c.IncResume("p") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./pkg/metrics/ -run TestPipelineCollector -v
```
Expected: compilation error.

- [ ] **Step 3: Create pipeline.go**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

type PipelineCollector struct {
	runTotal     *prometheus.CounterVec
	runDuration  *prometheus.HistogramVec
	stepTotal    *prometheus.CounterVec
	stepDuration *prometheus.HistogramVec
	stepRetry    *prometheus.CounterVec
	resumeTotal  *prometheus.CounterVec
}

func NewPipelineCollector(stats *stats.Stats) *PipelineCollector {
	if stats == nil {
		return &PipelineCollector{}
	}
	return &PipelineCollector{
		runTotal:     stats.RegisterCounterVec("pipeline_run_total", "Runs by pipeline and status", "pipeline", "status"),
		runDuration:  stats.RegisterHistogramVec("pipeline_run_duration_seconds", "Run duration distribution", "pipeline", "status"),
		stepTotal:    stats.RegisterCounterVec("pipeline_step_total", "Steps by pipeline, step, and status", "pipeline", "step", "status"),
		stepDuration: stats.RegisterHistogramVec("pipeline_step_duration_seconds", "Step duration distribution", "pipeline", "step", "capability", "status"),
		stepRetry:    stats.RegisterCounterVec("pipeline_step_retry_total", "Step retry count", "pipeline", "step"),
		resumeTotal:  stats.RegisterCounterVec("pipeline_resume_total", "Pipeline resume count", "pipeline"),
	}
}

func (c *PipelineCollector) IncRunTotal(pipeline, status string) {
	if c.runTotal == nil { return }
	defer recoverLog("pipeline_run_total")
	c.runTotal.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(status)).Inc()
}

func (c *PipelineCollector) ObserveRunDuration(pipeline, status string, seconds float64) {
	if c.runDuration == nil { return }
	defer recoverLog("pipeline_run_duration_seconds")
	c.runDuration.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(status)).Observe(seconds)
}

func (c *PipelineCollector) IncStepTotal(pipeline, step, status string) {
	if c.stepTotal == nil { return }
	defer recoverLog("pipeline_step_total")
	c.stepTotal.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(step), sanitizeLabel(status)).Inc()
}

func (c *PipelineCollector) ObserveStepDuration(pipeline, step, capability, status string, seconds float64) {
	if c.stepDuration == nil { return }
	defer recoverLog("pipeline_step_duration_seconds")
	c.stepDuration.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(step), sanitizeLabel(capability), sanitizeLabel(status)).Observe(seconds)
}

func (c *PipelineCollector) IncStepRetry(pipeline, step string) {
	if c.stepRetry == nil { return }
	defer recoverLog("pipeline_step_retry_total")
	c.stepRetry.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(step)).Inc()
}

func (c *PipelineCollector) IncResume(pipeline string) {
	if c.resumeTotal == nil { return }
	defer recoverLog("pipeline_resume_total")
	c.resumeTotal.WithLabelValues(sanitizeLabel(pipeline)).Inc()
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/metrics/ -run TestPipelineCollector -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/pipeline.go pkg/metrics/pipeline_test.go
git commit -m "feat(metrics): add PipelineCollector"
```

---

### Task 4: Create `WorkflowCollector`

**Files:**
- Create: `pkg/metrics/workflow.go`
- Create: `pkg/metrics/workflow_test.go`

- [ ] **Step 1: Write failing test**

`pkg/metrics/workflow_test.go`:
```go
package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewWorkflowCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewWorkflowCollector(nil)
		assert.NotNil(t, c)
		c.IncRunTotal("w", "done")
		c.ObserveRunDuration("w", "done", 2.0)
		c.IncStepTotal("w", "s", "done")
		c.ObserveStepDuration("w", "s", "capability", "done", 0.5)
		c.IncStepRetry("w", "s")
		c.IncResume("w")
		c.SetConcurrency("w", 3)
	})
}

func TestWorkflowCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewWorkflowCollector(s)

	c.IncRunTotal("archive-workflow", "done")
	c.IncRunTotal("archive-workflow", "done")
	c.IncRunTotal("archive-workflow", "failed")

	c.IncStepRetry("archive-workflow", "task1")
	c.IncStepRetry("archive-workflow", "task1")
	c.IncStepRetry("archive-workflow", "task2")

	c.IncResume("archive-workflow")

	expected := `
# HELP workflow_run_total Runs by workflow and status
# TYPE workflow_run_total counter
workflow_run_total{workflow="archive-workflow",status="done"} 2
workflow_run_total{workflow="archive-workflow",status="failed"} 1
`
	err := testutil.CollectAndCompare(c.runTotal, strings.NewReader(expected), "workflow_run_total")
	assert.NoError(t, err)
}

func TestWorkflowCollector_ConcurrencyGauge(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewWorkflowCollector(s)
	c.SetConcurrency("dag-workflow", 3)
	c.SetConcurrency("dag-workflow", 0)
	assert.NotNil(t, c.concurrency)
}

func TestWorkflowCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewWorkflowCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncRunTotal", fn: func() { c.IncRunTotal("w", "done") }},
		{name: "ObserveRunDuration", fn: func() { c.ObserveRunDuration("w", "done", 1.0) }},
		{name: "IncStepTotal", fn: func() { c.IncStepTotal("w", "s", "done") }},
		{name: "SetConcurrency", fn: func() { c.SetConcurrency("w", 5) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./pkg/metrics/ -run TestWorkflowCollector -v
```
Expected: compilation error.

- [ ] **Step 3: Create workflow.go**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

type WorkflowCollector struct {
	runTotal     *prometheus.CounterVec
	runDuration  *prometheus.HistogramVec
	stepTotal    *prometheus.CounterVec
	stepDuration *prometheus.HistogramVec
	stepRetry    *prometheus.CounterVec
	resumeTotal  *prometheus.CounterVec
	concurrency  *prometheus.GaugeVec
}

func NewWorkflowCollector(stats *stats.Stats) *WorkflowCollector {
	if stats == nil {
		return &WorkflowCollector{}
	}
	return &WorkflowCollector{
		runTotal:     stats.RegisterCounterVec("workflow_run_total", "Runs by workflow and status", "workflow", "status"),
		runDuration:  stats.RegisterHistogramVec("workflow_run_duration_seconds", "Run duration distribution", "workflow", "status"),
		stepTotal:    stats.RegisterCounterVec("workflow_step_total", "Steps by workflow, step, and status", "workflow", "step", "status"),
		stepDuration: stats.RegisterHistogramVec("workflow_step_duration_seconds", "Step duration distribution", "workflow", "step", "action_type", "status"),
		stepRetry:    stats.RegisterCounterVec("workflow_step_retry_total", "Step retry count", "workflow", "step"),
		resumeTotal:  stats.RegisterCounterVec("workflow_resume_total", "Workflow resume count", "workflow"),
		concurrency:  stats.RegisterGaugeVec("workflow_concurrency_gauge", "Running tasks in DAG parallel mode", "workflow"),
	}
}

func (c *WorkflowCollector) IncRunTotal(workflow, status string) {
	if c.runTotal == nil { return }
	defer recoverLog("workflow_run_total")
	c.runTotal.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(status)).Inc()
}

func (c *WorkflowCollector) ObserveRunDuration(workflow, status string, seconds float64) {
	if c.runDuration == nil { return }
	defer recoverLog("workflow_run_duration_seconds")
	c.runDuration.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(status)).Observe(seconds)
}

func (c *WorkflowCollector) IncStepTotal(workflow, step, status string) {
	if c.stepTotal == nil { return }
	defer recoverLog("workflow_step_total")
	c.stepTotal.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(step), sanitizeLabel(status)).Inc()
}

func (c *WorkflowCollector) ObserveStepDuration(workflow, step, actionType, status string, seconds float64) {
	if c.stepDuration == nil { return }
	defer recoverLog("workflow_step_duration_seconds")
	c.stepDuration.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(step), sanitizeLabel(actionType), sanitizeLabel(status)).Observe(seconds)
}

func (c *WorkflowCollector) IncStepRetry(workflow, step string) {
	if c.stepRetry == nil { return }
	defer recoverLog("workflow_step_retry_total")
	c.stepRetry.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(step)).Inc()
}

func (c *WorkflowCollector) IncResume(workflow string) {
	if c.resumeTotal == nil { return }
	defer recoverLog("workflow_resume_total")
	c.resumeTotal.WithLabelValues(sanitizeLabel(workflow)).Inc()
}

func (c *WorkflowCollector) SetConcurrency(workflow string, count int) {
	if c.concurrency == nil { return }
	defer recoverLog("workflow_concurrency_gauge")
	c.concurrency.WithLabelValues(sanitizeLabel(workflow)).Set(float64(count))
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/metrics/ -run TestWorkflowCollector -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/workflow.go pkg/metrics/workflow_test.go
git commit -m "feat(metrics): add WorkflowCollector"
```

---

### Task 5: Create `EventCollector`

**Files:**
- Create: `pkg/metrics/event.go`
- Create: `pkg/metrics/event_test.go`

- [ ] **Step 1: Write failing test**

`pkg/metrics/event_test.go`:
```go
package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewEventCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewEventCollector(nil)
		assert.NotNil(t, c)
		c.IncReceived("bookmark.created", "ability")
		c.IncMatched("bookmark.created", "archive-items")
		c.IncDedup("bookmark.created", "archive-items")
		c.ObserveLag("bookmark.created", 0.5)
	})
}

func TestEventCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewEventCollector(s)

	c.IncReceived("bookmark.created", "ability")
	c.IncReceived("bookmark.created", "ability")
	c.IncReceived("kanban.task.created", "ability")

	c.IncMatched("bookmark.created", "archive-items")
	c.IncMatched("bookmark.created", "sync-bookmarks")

	c.IncDedup("bookmark.created", "archive-items")

	c.ObserveLag("bookmark.created", 1.2)

	expected := `
# HELP event_received_total Events received by event type and source
# TYPE event_received_total counter
event_received_total{event_type="bookmark.created",source="ability"} 2
event_received_total{event_type="kanban.task.created",source="ability"} 1
`
	err := testutil.CollectAndCompare(c.receivedTotal, strings.NewReader(expected), "event_received_total")
	assert.NoError(t, err)
}

func TestEventCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewEventCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncReceived", fn: func() { c.IncReceived("e", "s") }},
		{name: "IncMatched", fn: func() { c.IncMatched("e", "p") }},
		{name: "IncDedup", fn: func() { c.IncDedup("e", "p") }},
		{name: "ObserveLag", fn: func() { c.ObserveLag("e", 1.0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./pkg/metrics/ -run TestEventCollector -v
```
Expected: compilation error.

- [ ] **Step 3: Create event.go**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

type EventCollector struct {
	receivedTotal *prometheus.CounterVec
	matchedTotal  *prometheus.CounterVec
	dedupTotal    *prometheus.CounterVec
	lagSeconds    *prometheus.HistogramVec
}

func NewEventCollector(stats *stats.Stats) *EventCollector {
	if stats == nil {
		return &EventCollector{}
	}
	return &EventCollector{
		receivedTotal: stats.RegisterCounterVec("event_received_total", "Events received by event type and source", "event_type", "source"),
		matchedTotal:  stats.RegisterCounterVec("event_matched_total", "Events matched to a pipeline", "event_type", "pipeline"),
		dedupTotal:    stats.RegisterCounterVec("event_dedup_total", "Idempotent consumption filter hits", "event_type", "pipeline"),
		lagSeconds:    stats.RegisterHistogramVec("event_lag_seconds", "Delay from event creation to consumption", "event_type"),
	}
}

func (c *EventCollector) IncReceived(eventType, source string) {
	if c.receivedTotal == nil { return }
	defer recoverLog("event_received_total")
	c.receivedTotal.WithLabelValues(sanitizeLabel(eventType), sanitizeLabel(source)).Inc()
}

func (c *EventCollector) IncMatched(eventType, pipeline string) {
	if c.matchedTotal == nil { return }
	defer recoverLog("event_matched_total")
	c.matchedTotal.WithLabelValues(sanitizeLabel(eventType), sanitizeLabel(pipeline)).Inc()
}

func (c *EventCollector) IncDedup(eventType, pipeline string) {
	if c.dedupTotal == nil { return }
	defer recoverLog("event_dedup_total")
	c.dedupTotal.WithLabelValues(sanitizeLabel(eventType), sanitizeLabel(pipeline)).Inc()
}

func (c *EventCollector) ObserveLag(eventType string, seconds float64) {
	if c.lagSeconds == nil { return }
	defer recoverLog("event_lag_seconds")
	c.lagSeconds.WithLabelValues(sanitizeLabel(eventType)).Observe(seconds)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/metrics/ -run TestEventCollector -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/event.go pkg/metrics/event_test.go
git commit -m "feat(metrics): add EventCollector"
```

---

### Task 6: Create `AbilityCollector`

**Files:**
- Create: `pkg/metrics/ability.go`
- Create: `pkg/metrics/ability_test.go`

- [ ] **Step 1: Write failing test**

`pkg/metrics/ability_test.go`:
```go
package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewAbilityCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewAbilityCollector(nil)
		assert.NotNil(t, c)
		c.IncInvokeTotal("bookmark", "list", "ok")
		c.ObserveInvokeDuration("bookmark", "list", 0.5)
		c.IncInvokeError("bookmark", "list", "timeout")
	})
}

func TestAbilityCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewAbilityCollector(s)

	c.IncInvokeTotal("bookmark", "list", "ok")
	c.IncInvokeTotal("bookmark", "list", "ok")
	c.IncInvokeTotal("bookmark", "create", "ok")
	c.IncInvokeTotal("kanban", "list", "ok")

	expected := `
# HELP ability_invoke_total Invocations by capability, operation, and status
# TYPE ability_invoke_total counter
ability_invoke_total{capability="bookmark",operation="list",status="ok"} 2
ability_invoke_total{capability="bookmark",operation="create",status="ok"} 1
ability_invoke_total{capability="kanban",operation="list",status="ok"} 1
`
	err := testutil.CollectAndCompare(c.invokeTotal, strings.NewReader(expected), "ability_invoke_total")
	assert.NoError(t, err)
}

func TestAbilityCollector_ErrorMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewAbilityCollector(s)

	c.IncInvokeError("bookmark", "list", "timeout")
	c.IncInvokeError("bookmark", "list", "timeout")
	c.IncInvokeError("bookmark", "list", "rate_limited")

	expected := `
# HELP ability_invoke_error_total Invocation errors by capability, operation, and error code
# TYPE ability_invoke_error_total counter
ability_invoke_error_total{capability="bookmark",operation="list",error_code="timeout"} 2
ability_invoke_error_total{capability="bookmark",operation="list",error_code="rate_limited"} 1
`
	err := testutil.CollectAndCompare(c.invokeErrorTotal, strings.NewReader(expected), "ability_invoke_error_total")
	assert.NoError(t, err)
}

func TestAbilityCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewAbilityCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncInvokeTotal", fn: func() { c.IncInvokeTotal("c", "o", "ok") }},
		{name: "ObserveInvokeDuration", fn: func() { c.ObserveInvokeDuration("c", "o", 1.0) }},
		{name: "IncInvokeError", fn: func() { c.IncInvokeError("c", "o", "err") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./pkg/metrics/ -run TestAbilityCollector -v
```
Expected: compilation error.

- [ ] **Step 3: Create ability.go**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

type AbilityCollector struct {
	invokeTotal      *prometheus.CounterVec
	invokeDuration   *prometheus.HistogramVec
	invokeErrorTotal *prometheus.CounterVec
}

func NewAbilityCollector(stats *stats.Stats) *AbilityCollector {
	if stats == nil {
		return &AbilityCollector{}
	}
	return &AbilityCollector{
		invokeTotal:      stats.RegisterCounterVec("ability_invoke_total", "Invocations by capability, operation, and status", "capability", "operation", "status"),
		invokeDuration:   stats.RegisterHistogramVec("ability_invoke_duration_seconds", "Invocation duration distribution", "capability", "operation"),
		invokeErrorTotal: stats.RegisterCounterVec("ability_invoke_error_total", "Invocation errors by capability, operation, and error code", "capability", "operation", "error_code"),
	}
}

func (c *AbilityCollector) IncInvokeTotal(capability, operation, status string) {
	if c.invokeTotal == nil { return }
	defer recoverLog("ability_invoke_total")
	c.invokeTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(status)).Inc()
}

func (c *AbilityCollector) ObserveInvokeDuration(capability, operation string, seconds float64) {
	if c.invokeDuration == nil { return }
	defer recoverLog("ability_invoke_duration_seconds")
	c.invokeDuration.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation)).Observe(seconds)
}

func (c *AbilityCollector) IncInvokeError(capability, operation, errorCode string) {
	if c.invokeErrorTotal == nil { return }
	defer recoverLog("ability_invoke_error_total")
	c.invokeErrorTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(errorCode)).Inc()
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/metrics/ -run TestAbilityCollector -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/ability.go pkg/metrics/ability_test.go
git commit -m "feat(metrics): add AbilityCollector"
```

---

### Task 7: Create Fx module file

**Files:**
- Create: `pkg/metrics/metrics.go`

- [ ] **Step 1: Create metrics.go**

```go
package metrics

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func Module() fx.Option {
	return fx.Module("metrics",
		fx.Provide(
			stats.NewStats,
			NewPipelineCollector,
			NewWorkflowCollector,
			NewEventCollector,
			NewAbilityCollector,
		),
	)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./pkg/metrics/...
```
Expected: builds without errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/metrics/metrics.go
git commit -m "feat(metrics): add Fx module"
```

---

### Task 8: Add `CreatedAt` to `DataEvent`

**Files:**
- Modify: `pkg/types/event.go`
- Create: `pkg/types/event_test.go`

- [ ] **Step 1: Write failing test**

`pkg/types/event_test.go`:
```go
package types

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataEventCreatedAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    DataEvent
	}{
		{
			name: "default zero time serialized and restored",
			d:    DataEvent{EventID: "evt-1", EventType: "test.event"},
		},
		{
			name: "explicit created at preserved",
			d: DataEvent{
				EventID:   "evt-2",
				EventType: "test.event",
				CreatedAt: time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "round-trip serialization",
			d: DataEvent{
				EventID:   "evt-3",
				EventType: "test.event",
				CreatedAt: time.Now().Truncate(time.Second),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := sonic.Marshal(tt.d)
			require.NoError(t, err)

			var restored DataEvent
			err = sonic.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, tt.d.EventID, restored.EventID)
			assert.Equal(t, tt.d.EventType, restored.EventType)
			if tt.d.CreatedAt.IsZero() {
				assert.True(t, restored.CreatedAt.IsZero())
			} else {
				assert.True(t, tt.d.CreatedAt.Equal(restored.CreatedAt))
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./pkg/types/ -run TestDataEventCreatedAt -v
```
Expected: compilation error (CreatedAt field doesn't exist).

- [ ] **Step 3: Add CreatedAt field**

In `pkg/types/event.go`, add `"time"` to imports. Update DataEvent struct:

```go
type DataEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Source         string    `json:"source"`
	Capability     string    `json:"capability"`
	Operation      string    `json:"operation"`
	Backend        string    `json:"backend"`
	App            string    `json:"app"`
	EntityID       string    `json:"entity_id"`
	CreatedAt      time.Time `json:"created_at"`
	IdempotencyKey string    `json:"idempotency_key"`
	UID            string    `json:"uid"`
	Topic          string    `json:"topic"`
	Data           KV        `json:"data"`
}
```

- [ ] **Step 4: Run test to verify pass**

```bash
go test ./pkg/types/ -run TestDataEventCreatedAt -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/types/event.go pkg/types/event_test.go
git commit -m "feat(types): add CreatedAt field to DataEvent for event lag tracking"
```

---

### Task 9: Instrument Pipeline Engine

**Files:**
- Modify: `pkg/pipeline/engine.go`
- Modify: `pkg/pipeline/pipeline_test.go`
- Modify: `internal/server/pipeline.go`

- [ ] **Step 1: Add metrics fields to Engine and update NewEngine**

In `pkg/pipeline/engine.go`, add import:
```go
	"github.com/flowline-io/flowbot/pkg/metrics"
```

Update Engine struct:
```go
type Engine struct {
	defs         []Definition
	store        RunStore
	metrics      *metrics.PipelineCollector
	eventMetrics *metrics.EventCollector
	handler      func(ctx context.Context, event types.DataEvent) error
}
```

Update NewEngine:
```go
func NewEngine(defs []Definition, store RunStore, pc *metrics.PipelineCollector, ec *metrics.EventCollector) *Engine {
	e := &Engine{
		defs:         defs,
		store:        store,
		metrics:      pc,
		eventMetrics: ec,
	}
	e.handler = e.handleEvent
	return e
}
```

- [ ] **Step 1.5: Add event matched/dedup in handleEvent and executePipeline**

In `handleEvent` (around lines 76-89, inside the for loop iterating matched defs), add matched counter:
```go
	for _, def := range matched {
		if e.eventMetrics != nil {
			e.eventMetrics.IncMatched(event.EventType, def.Name)
		}
		if err := e.executePipeline(ctx, def, event); err != nil {
			flog.Error(fmt.Errorf("pipeline %s: %w", def.Name, err))
		}
	}
```

In `executePipeline`, after the idempotency check's early return (around line 106, after `HasConsumed` returns true):
```go
		if consumed {
			flog.Info("pipeline %s already consumed event %s", def.Name, event.EventID)
			if e.eventMetrics != nil {
				e.eventMetrics.IncDedup(event.EventType, def.Name)
			}
			return nil
		}
```

- [ ] **Step 2: Add instrumentation to executePipeline**

After `defer span.End()` (line 97), add:
```go
	runStart := time.Now()
```

After the for loop ends (before `e.finishRunRecord`, around line 142), add:
```go
	if e.metrics != nil {
		status := "done"
		if finalErr != nil {
			status = "cancel"
		}
		e.metrics.IncRunTotal(def.Name, status)
		e.metrics.ObserveRunDuration(def.Name, status, time.Since(runStart).Seconds())
	}
```

- [ ] **Step 3: Add instrumentation to executeStep**

After `defer span.End()` (line 157), add:
```go
	stepStart := time.Now()
```

After `bo := retryCfg.BuildBackOff()` (line 180), add:
```go
	if e.metrics != nil {
		e.metrics.IncStepTotal(pipelineName, step.Name, "start")
	}
```

In the success block (replace lines 184-189):
```go
		if err == nil {
			stepResult := extractResult(res)
			rc.RecordStepResult(step.Name, stepResult)
			e.updateStepRunRecord(stepRunID, model.PipelineDone, stepResult, "", attempt)
			flog.Info("pipeline %s step %s completed (attempt %d)", pipelineName, step.Name, attempt)

			if e.metrics != nil {
				e.metrics.IncStepTotal(pipelineName, step.Name, "done")
				e.metrics.ObserveStepDuration(pipelineName, step.Name, string(step.Capability), "done", time.Since(stepStart).Seconds())
				if attempt > 1 {
					e.metrics.IncStepRetry(pipelineName, step.Name)
				}
			}
			return nil
		}
```

In the non-retryable error block (replace lines 193-196):
```go
		if !retryCfg.RetryEnabled() || !isRetryable(err, retryCfg) {
			e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, err.Error(), attempt)

			if e.metrics != nil {
				e.metrics.IncStepTotal(pipelineName, step.Name, "cancel")
				e.metrics.ObserveStepDuration(pipelineName, step.Name, string(step.Capability), "cancel", time.Since(stepStart).Seconds())
				if attempt > 1 {
					e.metrics.IncStepRetry(pipelineName, step.Name)
				}
			}
			return fmt.Errorf("step %s: %w", step.Name, err)
		}
```

In the retries exhausted block (replace lines 199-202):
```go
		if nextDelay == backoff.Stop {
			e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, err.Error(), attempt)

			if e.metrics != nil {
				e.metrics.IncStepTotal(pipelineName, step.Name, "cancel")
				e.metrics.ObserveStepDuration(pipelineName, step.Name, string(step.Capability), "cancel", time.Since(stepStart).Seconds())
				if attempt > 1 {
					e.metrics.IncStepRetry(pipelineName, step.Name)
				}
			}
			return fmt.Errorf("step %s (retries exhausted): %w", step.Name, err)
		}
```

In the context-cancelled block (replace lines 208-212):
```go
		case <-ctx.Done():
			e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, ctx.Err().Error(), attempt)

			if e.metrics != nil {
				e.metrics.IncStepTotal(pipelineName, step.Name, "cancel")
				e.metrics.ObserveStepDuration(pipelineName, step.Name, string(step.Capability), "cancel", time.Since(stepStart).Seconds())
				if attempt > 1 {
					e.metrics.IncStepRetry(pipelineName, step.Name)
				}
			}
			return fmt.Errorf("step %s cancelled: %w", step.Name, ctx.Err())
```

- [ ] **Step 4: Add instrumentation to ResumePipeline**

After `GetRun` succeeds (after line 361), add:
```go
	if e.metrics != nil {
		e.metrics.IncResume(run.PipelineName)
	}
```

- [ ] **Step 5: Update pipeline_test.go**

Add import `"github.com/flowline-io/flowbot/pkg/metrics"`.

Update TestNewEngine to pass no-op collectors (PipelineCollector + EventCollector). Each `NewEngine` call gets 4 args now:

```go
for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		t.Parallel()
		noopPC := metrics.NewPipelineCollector(nil)
		noopEC := metrics.NewEventCollector(nil)
		e := NewEngine(tt.defs, tt.store, noopPC, noopEC)
		assert.NotNil(t, e)
		assert.NotNil(t, e.Handler())
	})
}
```

- [ ] **Step 6: Update initPipeline in server/pipeline.go**

Add import `"github.com/flowline-io/flowbot/pkg/metrics"`.

Replace the engine creation line (line 38) — note: nil for both collectors for now, full wiring via DI in Task 12:
```go
	engine := pipeline.NewEngine(pipelineDefs, runStore, metrics.NewPipelineCollector(nil), metrics.NewEventCollector(nil))
```

- [ ] **Step 7: Run tests

```bash
go test ./pkg/pipeline/ -v
go build ./internal/server/...
```
Expected: PASS, builds.

Check for other call sites of `NewEngine`:
```bash
rg "NewEngine\(" --type go -l
```
Update any additional call sites.

- [ ] **Step 8: Commit**

```bash
git add pkg/pipeline/engine.go pkg/pipeline/pipeline_test.go internal/server/pipeline.go
git commit -m "feat(pipeline): inject PipelineCollector and instrument run/step/resume"
```

---

### Task 10: Instrument Workflow Runner

**Files:**
- Modify: `pkg/workflow/workflow.go`
- Modify: `pkg/workflow/scheduler.go`
- Modify: `pkg/workflow/workflow_test.go`
- Modify: `pkg/workflow/scheduler_test.go`
- Any files calling `NewRunnerWithStore`

- [ ] **Step 1: Add metrics field to Runner and update constructors**

In `pkg/workflow/workflow.go`, add import:
```go
	"github.com/flowline-io/flowbot/pkg/metrics"
```

Update Runner struct:
```go
type Runner struct {
	engines      map[string]*executor.Engine
	store        WorkflowRunStore
	metrics      *metrics.WorkflowCollector
	workflowFile string
	triggerType  string
}
```

Update NewRunnerWithStore:
```go
func NewRunnerWithStore(store WorkflowRunStore, wc *metrics.WorkflowCollector, workflowFile, triggerType string) *Runner {
	return &Runner{
		engines: map[string]*executor.Engine{
			runtime.Capability: executor.New(runtime.Capability),
			runtime.Shell:      executor.New(runtime.Shell),
			runtime.Docker:     executor.New(runtime.Docker),
			runtime.Machine:    executor.New(runtime.Machine),
		},
		store:        store,
		metrics:      wc,
		workflowFile: workflowFile,
		triggerType:  triggerType,
	}
}
```

Update NewRunner:
```go
func NewRunner() *Runner {
	return NewRunnerWithStore(nil, nil, "", "")
}
```

- [ ] **Step 2: Instrument runSequential**

At the top of `runSequential` (after line 250), add:
```go
	start := time.Now()
	var runErr error
	defer func() {
		if r.metrics != nil {
			status := "done"
			if runErr != nil {
				status = "failed"
			}
			r.metrics.IncRunTotal(wf.Name, status)
			r.metrics.ObserveRunDuration(wf.Name, status, time.Since(start).Seconds())
		}
	}()
```

Replace all `return err` in the function body with `runErr = err; return`. In runSequential's step loop, five return locations exist:

(1) Line 274 — task not found. Before `return err`:
```go
			runErr = err
			return
```
(2) Line 283 — resolve params error. Before `return err`:
```go
			runErr = err
			return
```
(3) Line 302 — mapper step error. Before `return merr`:
```go
			runErr = merr
			return
```
(4) Line 323 — convert task error. Before `return err`:
```go
			runErr = err
			return
```
(5) Line 332 — step failure. Before `return fmt.Errorf(...)`:
```go
			runErr = fmt.Errorf("step %s failed: %w", stepID, rerr)
			return
```

Before the `r.runWithRetry` call (around line 326), add step start:
```go
		flog.Info("[workflow] running step %s: %s", stepID, wt.Action)
		stepStart := time.Now()
		if r.metrics != nil {
			r.metrics.IncStepTotal(wf.Name, stepID, "running")
		}

		attempt, rerr := r.runWithRetry(ctx, task, wt.Retry, stepID, stepRun)
```

After `runWithRetry` (line 328-333), update the error branch:
```go
		attempt, rerr := r.runWithRetry(ctx, task, wt.Retry, stepID, stepRun)
		if rerr != nil {
			r.failStep(stepRun, rerr, attempt)
			r.failRun(run, cancelHeartbeat, rerr)
			if r.metrics != nil {
				r.metrics.IncStepTotal(wf.Name, stepID, "failed")
				r.metrics.ObserveStepDuration(wf.Name, stepID, info.Type, "failed", time.Since(stepStart).Seconds())
				if attempt > 1 {
					r.metrics.IncStepRetry(wf.Name, stepID)
				}
			}
			runErr = fmt.Errorf("step %s failed: %w", stepID, rerr)
			return
		}
```

And after step success (after stepIndex++ at line 349), add:
```go
		if r.metrics != nil {
			r.metrics.IncStepTotal(wf.Name, stepID, "done")
			r.metrics.ObserveStepDuration(wf.Name, stepID, info.Type, "done", time.Since(stepStart).Seconds())
			if attempt > 1 {
				r.metrics.IncStepRetry(wf.Name, stepID)
			}
		}
```
Note: `info` is already computed at line 286 (`info := ParseAction(wt.Action)`), accessible throughout the iteration.

For mapper steps (lines 296-313), add step timing before the mapper block:
```go
		if info.Type == "mapper" {
			stepStart := time.Now()
			if r.metrics != nil {
				r.metrics.IncStepTotal(wf.Name, stepID, "running")
			}
			// ... existing mapper code (lines 297-312) ...
			if r.metrics != nil {
				r.metrics.IncStepTotal(wf.Name, stepID, "done")
				r.metrics.ObserveStepDuration(wf.Name, stepID, "mapper", "done", time.Since(stepStart).Seconds())
				// mapper steps don't have retries, no IncStepRetry
			}
			stepIndex++
			continue
		}
```

For the error return paths (lines 272-284, 297-303, 319-324) that don't go through runWithRetry (they fail before running), add step-level failure metrics with status "failed" before each early `runErr = err; return`.

- [ ] **Step 3: Instrument runParallel in scheduler.go**

Add import `"time"` and `"github.com/flowline-io/flowbot/pkg/metrics"` to scheduler.go.

At the top of `runParallel` (after line 54), add:
```go
	parallelStart := time.Now()
	var finalErr error
	defer func() {
		if r.metrics != nil {
			status := "done"
			if finalErr != nil {
				status = "failed"
			}
			r.metrics.IncRunTotal(wf.Name, status)
			r.metrics.ObserveRunDuration(wf.Name, status, time.Since(parallelStart).Seconds())
		}
	}()
```

In the task dispatch inner loop (around line 86, inside `go func(taskID string)`), add step start and concurrency gauge after `go func(taskID string) {`:
```go
				go func(taskID string) {
					defer wg.Done()
					defer func() {
						<-sem
						done <- struct{}{}
					}()

					wt := taskMap[taskID]
					stepStart := time.Now()
					info := ParseAction(wt.Action)
					if r.metrics != nil {
						r.metrics.IncStepTotal(wf.Name, taskID, "running")
					}

					rerr := r.executeParallelTask(ctx, taskID, wt, nodes, input, &results, &mu, run, &ready, taskMap, &wf)
					if rerr != nil {
						errOnce.Do(func() {
							firstErr = rerr
							cancel()
						})
						if r.metrics != nil {
							r.metrics.IncStepTotal(wf.Name, taskID, "failed")
							r.metrics.ObserveStepDuration(wf.Name, taskID, info.Type, "failed", time.Since(stepStart).Seconds())
						}
						// retry count is tracked inside executeParallelTask via runEngineWithRetry
					} else {
						if r.metrics != nil {
							r.metrics.IncStepTotal(wf.Name, taskID, "done")
							r.metrics.ObserveStepDuration(wf.Name, taskID, info.Type, "done", time.Since(stepStart).Seconds())
						}
					}
				}(id)
```

For concurrency gauge, in the dispatch loop after activeCount increment (around line 82, before `go func`):
```go
				mu.Unlock()
				if r.metrics != nil {
					r.metrics.SetConcurrency(wf.Name, activeCount)
				}
```

In the done channel handler (around line 114-118), after decrementing activeCount:
```go
			mu.Unlock()
			if r.metrics != nil {
				r.metrics.SetConcurrency(wf.Name, activeCount)
			}
```

Replace `return firstErr` (line 128-132) with:
```go
	if firstErr != nil {
		if r.store != nil && run != nil {
			_ = r.store.UpdateRunStatus(run.ID, model.WorkflowRunFailed, firstErr.Error())
		}
		finalErr = firstErr
		return firstErr
	}
```
Note: the deferred run metrics closure captures `finalErr`.

In `executeParallelTask`, add step start timer at top (after line 155):
```go
	stepStart := time.Now()
```

In the error return after `r.runEngineWithRetry` (line 220), add step retry counter:
```go
		if rerr != nil {
			r.failStep(stepRun, rerr, attempt)
			if r.metrics != nil && attempt > 1 {
				r.metrics.IncStepRetry(wf.Name, taskID)
			}
			return fmt.Errorf("step %s failed: %w", taskID, rerr)
		}
```

In the success path (lin 238), add retry counter if attempt > 1. Since `attempt` is not in scope here (it's a local variable from the if block), record it inside the `if rerr != nil` check above or extract. Simplest: also check attempt inside the success path by moving the stepRun update above the success metrics:
```go
		if r.store != nil && stepRun != nil {
			resultJSON := model.JSON{}
			if task.Result != "" {
				resultRaw, _ := pooledSonic.Marshal(map[string]any{"result": task.Result})
				_ = resultJSON.Scan(resultRaw)
			}
			_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", attempt)
		}
		if r.metrics != nil && attempt > 1 {
			r.metrics.IncStepRetry(wf.Name, taskID)
		}
```
`attempt` is the return value from `r.runEngineWithRetry`, available at this point in the code.

- [ ] **Step 4: Instrument ResumeWorkflow**

In `ResumeWorkflow` (line 363), after loading the workflow (after `wf, err := LoadFile(run.WorkflowFile)` at line 379):
```go
	if r.metrics != nil {
		r.metrics.IncResume(wf.Name)
	}
```

- [ ] **Step 5: Update test files**

In `pkg/workflow/workflow_test.go` and `pkg/workflow/scheduler_test.go`, find all `NewRunnerWithStore` calls and add `nil` as the metrics parameter.

Search all call sites:
```bash
rg "NewRunnerWithStore\(" --type go -l
```

Update each file.

- [ ] **Step 6: Run tests**

```bash
go test ./pkg/workflow/ -v
go build ./...
```

Verify all call sites updated.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat(workflow): inject WorkflowCollector and instrument runSequential, runParallel, and resume"
```

---

### Task 11: Instrument Ability Invoke

**Files:**
- Modify: `pkg/ability/invoke.go`
- Modify: `pkg/ability/invoke_test.go`

- [ ] **Step 1: Add metrics field and global setter**

In `invoke.go`, add import:
```go
	"time"
	"github.com/flowline-io/flowbot/pkg/metrics"
```

Update `Registry` struct:
```go
type Registry struct {
	mu       sync.RWMutex
	handlers map[hub.CapabilityType]map[string]Invoker
	emitter  EventEmitter
	metrics  *metrics.AbilityCollector
}
```

Add after `SetEventEmitter`:
```go
func SetMetricsCollector(mc *metrics.AbilityCollector) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	DefaultRegistry.metrics = mc
}
```

- [ ] **Step 2: Instrument Invoke method**

In `Registry.Invoke`, add `start := time.Now()` before `result, err := invoker(ctx, params)` (before line 86).

On error (replacing lines 87-89):
```go
	if err != nil {
		trace.RecordError(ctx, err)
		r.mu.RLock()
		mc := r.metrics
		r.mu.RUnlock()
		if mc != nil {
			mc.IncInvokeTotal(string(capability), operation, "error")
			mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
			code := "unknown"
			if te, ok := err.(*types.Error); ok {
				code = te.Code
			}
			mc.IncInvokeError(string(capability), operation, code)
		}
		return nil, err
	}
```

On success (after result is set up, before the emitter goroutine at line 97):
```go
	r.mu.RLock()
	mc := r.metrics
	r.mu.RUnlock()
	if mc != nil {
		mc.IncInvokeTotal(string(capability), operation, "ok")
		mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
	}
```

- [ ] **Step 3: Update tests**

In `invoke_test.go`, add import:
```go
	"github.com/flowline-io/flowbot/pkg/metrics"
```

Add test:
```go
func TestSetMetricsCollector(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"sets nil collector without panic"},
		{"sets no-op collector"},
		{"can set after default registry is created"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() {
				SetMetricsCollector(nil)
				SetMetricsCollector(metrics.NewAbilityCollector(nil))
			})
		})
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/ability/ -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/invoke.go pkg/ability/invoke_test.go
git commit -m "feat(ability): inject AbilityCollector and instrument Invoke"
```

---

### Task 12: Wire Event Layer and Finalize Fx Integration

**Files:**
- Modify: `internal/server/pipeline.go`
- Modify: `internal/server/fx.go`

- [ ] **Step 1: Update initPipeline to receive all collectors**

In `internal/server/pipeline.go`, update signature:
```go
func initPipeline(
	lc fx.Lifecycle,
	cfg *config.Type,
	router *message.Router,
	subscriber message.Subscriber,
	pc *metrics.PipelineCollector,
	ec *metrics.EventCollector,
	ac *metrics.AbilityCollector,
) error {
```

- [ ] **Step 2: Pass collectors to NewEngine**

Replace the nil-passing call from Task 9:
```go
	engine := pipeline.NewEngine(pipelineDefs, runStore, pc, ec)
```

- [ ] **Step 3: Wire AbilityCollector**

After engine creation, before `ability.SetEventEmitter`:
```go
	ability.SetMetricsCollector(ac)
```

- [ ] **Step 4: Add EventCollector instrumentation in Watermill handler**

In the Watermill handler closure (around lines 82-89), add metric calls before processing:
```go
		func(msg *message.Message) error {
			var dataEvent types.DataEvent
			if err := sonic.Unmarshal(msg.Payload, &dataEvent); err != nil {
				return fmt.Errorf("unmarshal data event: %w", err)
			}

			if ec != nil {
				ec.IncReceived(dataEvent.EventType, dataEvent.Source)
				if !dataEvent.CreatedAt.IsZero() {
					ec.ObserveLag(dataEvent.EventType, time.Since(dataEvent.CreatedAt).Seconds())
				}
			}

			ctx, cancel := context.WithTimeout(msg.Context(), 10*time.Minute)
			defer cancel()
			return engine.Handler()(ctx, dataEvent)
		},
```

- [ ] **Step 5: Set CreatedAt in the event emitter**

In the event emitter (around line 55-65), add CreatedAt:
```go
			dataEvent := types.DataEvent{
				EventID:        eventID,
				EventType:      ref.EventType,
				Source:         "ability",
				Capability:     string(result.Capability),
				Operation:      result.Operation,
				Backend:        desc.Backend,
				App:            desc.App,
				EntityID:       ref.EntityID,
				IdempotencyKey: eventID,
				CreatedAt:      time.Now(),
			}
```

- [ ] **Step 6: Update internal/server/fx.go**

Add `metrics.Module()` to the Fx module options:
```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/metrics"
)

var Modules = fx.Options(
	modules.Modules,
	NotifyModules,
	MediaModules,
	metrics.Module(),
	fx.Provide(
		// ... existing providers ...
	),
	// ...
)
```

- [ ] **Step 7: Build and verify**

```bash
go build ./internal/server/...
go build ./...
go vet ./...
```
Expected: builds and vets without errors.

- [ ] **Step 8: Commit**

```bash
git add internal/server/pipeline.go internal/server/fx.go
git commit -m "feat(server): wire all metric collectors via Fx into pipeline, event, and ability layers"
```

---

### Task 13: Run full test suite and lint

- [ ] **Step 1: Run all unit tests**

```bash
go test ./pkg/stats/... ./pkg/metrics/... ./pkg/types/... ./pkg/pipeline/... ./pkg/workflow/... ./pkg/ability/... -v -count=1
```

- [ ] **Step 2: Run lint**

```bash
go tool task lint
```

- [ ] **Step 3: Fix any issues**

Address lint warnings and test failures.

- [ ] **Step 4: Run race tests**

```bash
go test ./pkg/metrics/... -race -count=1
```

- [ ] **Step 5: Commit fixes if any**

```bash
git add -A && git commit -m "chore: fix lint and test issues from metrics collector wiring"
```
