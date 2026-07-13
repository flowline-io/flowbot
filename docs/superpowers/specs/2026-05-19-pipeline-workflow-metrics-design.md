# Pipeline / Workflow Business Metrics Monitoring Design

**Date**: 2026-05-19
**Status**: Draft

---

## Overview

Add business-level metrics for pipeline execution, workflow execution, event processing, and ability invocation. Provide SRE-grade alerting dimensions (throughput, latency, error rate) and business reporting dimensions (execution counts, retry trends) within the existing Prometheus + Pushgateway infrastructure.

## Architecture

### Package Structure

```
pkg/metrics/
├── metrics.go          # Fx Module, WithStats() constructor
├── types.go            # shared label types, metric name constants
├── pipeline.go         # PipelineCollector (Counter/Gauge/Histogram)
├── workflow.go         # WorkflowCollector
├── event.go            # EventCollector (consumer-layer instrumentation)
├── capability.go          # AbilityCollector (invocation-layer instrumentation)
├── pipeline_test.go
├── workflow_test.go
├── event_test.go
└── ability_test.go
```

**Dependency direction**: `pkg/metrics/` -> `pkg/stats/` (bridges to Prometheus Pushgateway). No reverse dependency. Engine and ability packages receive collectors via constructor injection.

### pkg/stats Extension

`pkg/stats/` currently only supports flat `Counter`/`Gauge` via `Register(name, kind, help, labels...)` returning `*metricWrapper` (implements `MetricInterface` with `Inc/Add/Set`). For labeled vector metrics (`CounterVec`, `GaugeVec`, `HistogramVec`), add three new methods:

```go
func (s *Stats) RegisterCounterVec(name, help string, labelNames ...string) *prometheus.CounterVec
func (s *Stats) RegisterGaugeVec(name, help string, labelNames ...string) *prometheus.GaugeVec
func (s *Stats) RegisterHistogramVec(name, help string, labelNames ...string) *prometheus.HistogramVec
```

These register directly with `s.registry` (the `*prometheus.Registry`) and return the native Prometheus vec types. The existing Pushgateway interval push (`pushPeriodically`) already pushes the full registry, so vector metrics are included automatically without additional pushgateway changes. If `s == nil`, these panic (callers always check stats != nil before calling Register).

### Histogram Buckets

Use Prometheus `prometheus.DefBuckets` (`.005`, `.01`, `.025`, `.05`, `.1`, `.25`, `.5`, `1`, `2.5`, `5`, `10`) for all duration histograms. These cover sub-second API calls up to multi-minute pipeline runs.

### Internal Structure Per Collector

Each collector receives `*stats.Stats` in its constructor, calls the appropriate `Register*Vec` method, and holds the resulting `*prometheus.*Vec` reference. All public methods are nil-safe (no-op when collector is nil) and panic-safe (internal recover + log). Label values are sanitized: non `[a-zA-Z0-9_.-]` characters replaced with `_`, strings truncated at 128 chars.

```go
// pkg/metrics/pipeline.go (sketch)

type PipelineCollector struct {
    runTotal       *prometheus.CounterVec
    runDuration    *prometheus.HistogramVec
    stepTotal      *prometheus.CounterVec
    stepDuration   *prometheus.HistogramVec
    stepRetry      *prometheus.CounterVec
    resumeTotal    *prometheus.CounterVec
}

func NewPipelineCollector(stats *stats.Stats) *PipelineCollector {
    if stats == nil {
        return &PipelineCollector{}
    }
    return &PipelineCollector{
        runTotal:    stats.RegisterCounterVec("pipeline_run_total", "Runs by pipeline and status", "pipeline", "status"),
        runDuration: stats.RegisterHistogramVec("pipeline_run_duration_seconds", "Run duration distribution", "pipeline", "status"),
        // ...
    }
}

func (c *PipelineCollector) IncRunTotal(pipeline, status string) {
    if c.runTotal == nil { return }
    defer recoverLog("pipeline_run_total")
    c.runTotal.WithLabelValues(sanitize(pipeline), sanitize(status)).Inc()
}
```

## Metrics Definitions

### PipelineCollector

| Metric                           | Type      | Labels                                     | Description                                       |
| -------------------------------- | --------- | ------------------------------------------ | ------------------------------------------------- |
| `pipeline_run_total`             | Counter   | `pipeline`, `status`                       | Runs counted by status (start/done/cancel/failed) |
| `pipeline_run_duration_seconds`  | Histogram | `pipeline`, `status`                       | End-to-end run duration distribution              |
| `pipeline_step_total`            | Counter   | `pipeline`, `step`, `status`               | Steps counted by status                           |
| `pipeline_step_duration_seconds` | Histogram | `pipeline`, `step`, `capability`, `status` | Step duration distribution                        |
| `pipeline_step_retry_total`      | Counter   | `pipeline`, `step`                         | Step retry count                                  |
| `pipeline_resume_total`          | Counter   | `pipeline`                                 | Pipeline resume execution count                   |

### WorkflowCollector

| Metric                           | Type      | Labels                                      | Description                                 |
| -------------------------------- | --------- | ------------------------------------------- | ------------------------------------------- |
| `workflow_run_total`             | Counter   | `workflow`, `status`                        | Runs counted by status                      |
| `workflow_run_duration_seconds`  | Histogram | `workflow`, `status`                        | End-to-end run duration                     |
| `workflow_step_total`            | Counter   | `workflow`, `step`, `status`                | Steps counted by status                     |
| `workflow_step_duration_seconds` | Histogram | `workflow`, `step`, `action_type`, `status` | Step duration by action type                |
| `workflow_step_retry_total`      | Counter   | `workflow`, `step`                          | Step retry count                            |
| `workflow_resume_total`          | Counter   | `workflow`                                  | Resume execution count                      |
| `workflow_concurrency_gauge`     | Gauge     | `workflow`                                  | Currently running tasks (DAG parallel mode) |

### EventCollector

| Metric                 | Type      | Labels                   | Description                              |
| ---------------------- | --------- | ------------------------ | ---------------------------------------- |
| `event_received_total` | Counter   | `event_type`, `source`   | Total events received                    |
| `event_matched_total`  | Counter   | `event_type`, `pipeline` | Events matched to a pipeline             |
| `event_dedup_total`    | Counter   | `event_type`, `pipeline` | Idempotent consumption filter hits       |
| `event_lag_seconds`    | Histogram | `event_type`             | Delay from event creation to consumption |

### AbilityCollector

| Metric                            | Type      | Labels                                  | Description                      |
| --------------------------------- | --------- | --------------------------------------- | -------------------------------- |
| `ability_invoke_total`            | Counter   | `capability`, `operation`, `status`     | Invocation count by status       |
| `ability_invoke_duration_seconds` | Histogram | `capability`, `operation`               | Invocation duration distribution |
| `ability_invoke_error_total`      | Counter   | `capability`, `operation`, `error_code` | Error count by error code        |

**Total**: 19 metrics, 4 collectors.

## Instrumentation Points

### Pipeline Engine (`pkg/pipeline/engine.go`)

- **executePipeline()**: start timer at entry; after loop, record `pipeline_run_total` and `pipeline_run_duration_seconds`
- **executeStep()**: start timer per step; after ability call, record `pipeline_step_total`, `pipeline_step_duration_seconds`; if attempt > 1 record `pipeline_step_retry_total`
- **ResumePipeline()**: increment `pipeline_resume_total`

### Workflow Runner (`pkg/workflow/workflow.go`, `scheduler.go`)

- **runSequential() / runParallel()**: record `workflow_run_total` and `workflow_run_duration_seconds`; at resume increment `workflow_resume_total`
- **executeStep()**: record `workflow_step_total`, `workflow_step_duration_seconds`; if attempt > 1 record `workflow_step_retry_total`
- **runParallel()**: set/clear `workflow_concurrency_gauge` around goroutine launch/completion

### Event Layer (`internal/server/pipeline.go`, `pkg/event/pubsub.go`)

- **event_received_total**: in Watermill consumer middleware on topic `"pipeline:data_event"` receipt
- **event_matched_total**: in `Engine.handleEvent()` after `FindByEvent()` match success, per matched pipeline
- **event_dedup_total**: in `executePipeline()` when `HasConsumed()` returns true
- **event_lag_seconds**: DataEvent gains `CreatedAt time.Time` field, set by emitter; consumer computes `time.Since(event.CreatedAt)`

### Ability Layer (`pkg/ability/invoke.go`)

- In `Invoke()`: start timer; on completion record `ability_invoke_total` and `ability_invoke_duration_seconds`; on error record `ability_invoke_error_total`

### DataEvent Change

Add `CreatedAt time.Time` field to `pkg/types/event.go:DataEvent`. Set by emitter in `internal/server/pipeline.go` at event construction time.

## No-Op Fallback

When `metrics.enabled` is `false`, `*stats.Stats` is nil, and all collector constructors return zero-value collectors where every method is a no-op. Engine/ability code calls the same methods without nil checks.

```go
func NewNoopPipelineCollector() *PipelineCollector {
    return &PipelineCollector{}
}
```

All public methods use value receiver nil-safe pattern:

```go
func (c *PipelineCollector) IncRunTotal(pipeline, status string) {
    if c.runTotal == nil { return }
    // ...
}
```

## Configuration

```yaml
metrics:
  enabled:
    true # global switch: Prometheus endpoint, Pushgateway, and business collectors
    # false: all off, no-op collectors injected
```

No sub-switches for individual collector groups.

## Fx Dependency Injection

```go
// pkg/metrics/metrics.go
func Module() fx.Option {
    return fx.Module("metrics",
        fx.Provide(
            NewPipelineCollector,
            NewWorkflowCollector,
            NewEventCollector,
            NewAbilityCollector,
        ),
    )
}
```

Existing Fx provides updated with collector parameters:

- `pkg/pipeline/`: Engine constructor accepts `*PipelineCollector`
- `pkg/workflow/`: Runner constructor accepts `*WorkflowCollector`
- `pkg/ability/`: Registry constructor accepts `*AbilityCollector`
- `internal/server/pipeline.go`: Event handler closure captures `*EventCollector`

## Testing

### Unit Tests (TDD, co-located `_test.go`)

- Table-driven with `t.Run`, minimum 3 cases per table
- Use `prometheus/testutil.CollectAndCompare` to assert metric values
- Cover: normal inc/observe, label sanitize, no-op safety, concurrent access (`-race`), panic recover

### BDD Integration Tests (Ginkgo)

- Trigger a pipeline execution end-to-end, verify Pushgateway receives expected metric
- Trigger DAG parallel workflow, verify `workflow_concurrency_gauge` reflects active task count

### Impact on Existing Tests

- Pipeline engine tests: mock `PipelineCollector` interface or inject no-op
- Workflow runner tests: mock `WorkflowCollector` interface or inject no-op
- Ability invoke tests: mock `AbilityCollector` interface or inject no-op
- Event handler tests: mock `EventCollector` interface or inject no-op

No business behavior changes; only dependency injection wiring adjustments.

## Error Handling & Safety

- All collector methods: internal `defer recover()` + log, never propagate panic to caller
- Label sanitize: replace non `[a-zA-Z0-9_.-]` with `_`, truncate >128 chars
- Prometheus registration conflict: `GetOrCreate` pattern in `pkg/stats/` prevents duplicate registration
- Pushgateway unavailable: metrics accumulate locally, periodic push with existing 15s retry loop
