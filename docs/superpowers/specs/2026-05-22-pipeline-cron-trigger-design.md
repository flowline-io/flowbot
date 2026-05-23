# Pipeline Cron Trigger

Date: 2026-05-22

## Summary

Add cron-expresssion-based trigger to the pipeline engine so pipelines can
run on a schedule in addition to event-driven triggers.

## Motivation

Pipeline currently supports only event-based triggers (`Trigger.Event`).
Operators need scheduled pipelines (e.g. daily cleanup, periodic sync)
without an external event producer.

## Design

### Config layer (`pkg/config/config.go`)

`PipelineTrigger` gains `Cron` and `CronTimeout` fields:

```go
type PipelineTrigger struct {
    Event      string `json:"event" yaml:"event" mapstructure:"event"`
    Cron       string `json:"cron" yaml:"cron" mapstructure:"cron"`
    CronTimeout string `json:"cron_timeout" yaml:"cron_timeout" mapstructure:"cron_timeout"`
}
```

`CronTimeout` defaults to `"10m"` when empty. `Event` and `Cron` can coexist.
When both are set the pipeline triggers from either source.

**Enabled semantics**: `Pipeline.Enabled` controls ALL trigger types. An
`enabled: false` pipeline is never loaded by `LoadConfig` and is never
registered in the cron scheduler.

Example `pipelines.yaml`:

```yaml
- name: daily_cleanup
  description: "Daily cleanup job"
  enabled: true
  resumable: false
  trigger:
    cron: "0 3 * * *"
    cron_timeout: "30m"
  steps:
    - name: cleanup
      capability: system
      operation: cleanup
      params: {}
```

### Trigger model (`pkg/pipeline/loader.go`)

`Trigger` struct mirrors the config:

```go
type Trigger struct {
    Event      string
    Cron       string
    CronTimeout time.Duration
}
```

`LoadConfig` maps all fields. Additionally, `LoadConfig` validates cron
expressions by calling `cron.Parse` on each non-empty `Cron` field. Invalid
expressions cause the pipeline to be skipped with an error log (the
pipeline is not loaded).

### Cron expression format

Compatible with `github.com/flc1125/go-cron/v4` (already a project
dependency). Supports standard 5-field (`minute hour dom month dow`),
optional seconds, and descriptors (`@every 1h`, `@daily`).

Validation occurs at `LoadConfig` time so misconfigurations fail fast
before the scheduler starts.

### Engine-embedded scheduler (`pkg/pipeline/engine.go`)

- `Engine` holds `*cron.Cron` field.
- `NewEngine` creates the scheduler, registers cron-enabled definitions (`Enabled && Trigger.Cron != ""`), starts.
- `Engine.Stop()` stops the scheduler with a **30-second timeout**. After
  the timeout, in-flight jobs are force-cancelled and a warning is logged.

Each cron job runs independently:

1. **Acquires per-pipeline mutex** (see concurrency model below).
2. Creates context with `auth.SystemCronContext()` and the pipeline's
   configured `CronTimeout`.
3. Synthesizes a `DataEvent`:
   - `EventID`: `cron:<pipeline-name>:<unix-nano>-<randomHex(8)>` (UUID-level uniqueness, not clock-dependent)
   - `EventType`: `pipeline.cron:<pipeline-name>`
   - `Source`: `cron`
4. Calls `executePipeline(ctx, def, dataEvent)` — reuses the existing
   execution path including dedup, run records, checkpoints, metrics, and
   audit.

### Unified concurrency model

Concurrency control is centralized inside `executePipeline` via a
per-pipeline `sync.Mutex` map stored on `Engine`:

```
Engine.mu map[string]*sync.Mutex   // keyed by pipeline name
```

**All trigger sources** (event, cron, manual/resume) acquire the same
mutex for a given pipeline name before entering `executePipeline`. The
mutex is held for the full duration of pipeline execution. This guarantees:

- No two runs of the same pipeline overlap, regardless of trigger source.
- A cron tick while an event-triggered run is in-flight blocks until the
  first run completes.
- An event arriving while a cron run is in-flight blocks similarly.

The cron job uses `TryLock` (non-blocking) to skip a tick if the pipeline
is already running, logging a "skipped" metric. The Watermill event handler
uses `Lock` (blocking) so events are never silently dropped — they queue
up and execute when the previous run completes. Resume also uses `Lock`.

### Event ID uniqueness

The synthetic `EventID` uses nanosecond timestamp + 8 hex random:

```go
fmt.Sprintf("cron:%s:%d-%s", def.Name, time.Now().UnixNano(), randomHex(8))
```

This is immune to clock rollback and sub-millisecond collisions.

### Timeout configurability

Each cron-triggered pipeline can set `trigger.cron_timeout` (default
`"10m"`). The timeout applies to the context passed to `executePipeline`.
A nil or empty value defaults to 10 minutes.

### Dynamic reload (future)

The initial release does **not** support hot-reloading pipeline
definitions. The cron scheduler is populated once at `NewEngine` time. If
dynamic reload is needed later, a `Reload(defs []Definition)` method will
iterate existing cron entries, remove stale ones, and add new ones.

### Stop behavior

`Engine.Stop()`:

1. Calls `cron.Stop()` which returns a context that blocks until all
   running jobs complete OR the context is cancelled.
2. Wraps with `context.WithTimeout(ctx, 30*time.Second)`.
3. On timeout, logs a warning: "pipeline cron stop timed out, forcing
   shutdown" and returns.

### Observability

Cron-specific metrics on `PipelineCollector`:

| Metric                                       | Description                        |
| -------------------------------------------- | ---------------------------------- |
| `pipeline_cron_exec_total{pipeline, status}` | Cron runs by outcome (done/cancel) |
| `pipeline_cron_skip_total{pipeline}`         | Ticks skipped due to overlap       |
| `pipeline_cron_duration_seconds{pipeline}`   | Cron execution duration histogram  |

### Auth context

Cron-triggered pipeline steps use `auth.SystemCronContext()` so that
`ability.Invoke` calls are traceable to the cron subsystem.

## Key decisions

- **Mixed compatibility**: `event` and `cron` on the same trigger config
  are additive. A pipeline with both fires from either.
- **Synthetic DataEvent**: Cron does not emit events through Watermill / Redis
  Stream. It calls `executePipeline` directly with a fabricated event.
- **Unified concurrency**: Per-pipeline mutex in `executePipeline` protects
  ALL trigger sources (event, cron, manual/resume).
- **Enabled gate**: `enabled: false` prevents ALL triggering. Validated at
  `LoadConfig` time.
- **Cron validation at load time**: Invalid cron expressions skip the
  pipeline with an error log.
- **UUID-safe event IDs**: Nanosecond + random hex, not clock-dependent.
- **Configurable timeout**: Per-pipeline `cron_timeout`, default 10m.
- **Reuse execution path**: No new execution function. Cron shares
  `executePipeline`.
- **Dynamic reload deferred**: Not in initial scope.

## Known limitations

- Pipelines are static after `NewEngine`. Adding/removing pipelines
  requires a server restart.
- `Stop()` force-cancels jobs after 30s timeout, which may leave
  in-progress steps in an incomplete state (checkpointed if resumable).

## Testing

### Unit tests (TDD, table-driven)

All test functions use `for _, tt := range tests { t.Run(tt.name, ...) }`
with at least 3 cases per table. Happy path first, error cases required.

**`pkg/config/config_test.go`** -- parse cron fields from YAML:

- `tt.name`: "event only trigger", "cron only trigger", "both event and cron", "cron with custom timeout"
- verify `PipelineTrigger.Cron`, `PipelineTrigger.CronTimeout` parsed correctly

**`pkg/pipeline/loader_test.go`** -- `LoadConfig` maps cron and validates:

- `tt.name`: "event only definition", "cron only definition", "both triggers", "invalid cron expression skipped", "disabled pipeline skipped"
- verify `Trigger.Cron` mapped, invalid cron not loaded, `Enabled=false` not loaded

**`pkg/pipeline/engine_test.go`** -- cron engine behavior:

- `NewEngine` registers only enabled cron definitions: inject definitions with a
  testable clock (see clock abstraction below), verify only valid+enabled entries registered
- Unified concurrency: start an event-triggered run, fire a cron tick for
  same pipeline, verify cron is skipped (TryLock returns false), event run
  completes normally
- Concurrency across different pipelines: two pipelines with different
  names run concurrently without blocking each other
- `Stop()` timeout: create a blocking cron job, call `Stop()` with a short
  timeout in test, verify it returns after timeout with warning
- Synthetic event: verify `EventID` format, `EventType`, `Source`, and
  that `EventID` contains nanosecond and random suffix
- Cron metrics: verify counters increment for success, skip

#### Test clock abstraction

To keep tests deterministic and fast, `Engine` accepts an optional
`Clock` interface:

```go
type Clock interface {
    Now() time.Time
    After(d time.Duration) <-chan time.Time
}
```

A `RealClock` uses `time.Now` / `time.After`. A `FakeClock` in tests
allows manual time advancement, eliminating `time.Sleep` and flaky timing
assertions. BDD specs use `FakeClock` for reliable cron scheduling tests.

### BDD specs (Ginkgo v2 + Gomega)

`tests/specs/pipeline_spec_test.go` -- `Describe("Cron trigger")`:

- `It("executes pipeline on cron schedule")`: create pipeline with cron,
  use `FakeClock` to advance time past the schedule, assert runs complete
- `It("skips overlapping runs when cron fires twice")`: advance clock to
  fire cron while first run is in-flight, assert second tick skipped
- `It("blocks event trigger while cron is running")`: start cron run, send
  matching event, assert event run waits and executes after cron completes
- `It("stops execution on engine shutdown")`: start cron pipeline, call
  `Stop()`, advance clock, assert no more runs occur
- `It("supports mixed event and cron trigger")`: pipeline with both
  `event` and `cron`, assert both trigger paths work
- `It("records correct DataEvent for cron run")`: assert run records have
  `event_type` = `pipeline.cron:<name>` and `source` = `cron`
- `It("respects cron_timeout configuration")`: pipeline with short
  `cron_timeout` and long step, assert context deadline exceeded

## Files affected

| File                                | Change                                                                               |
| ----------------------------------- | ------------------------------------------------------------------------------------ |
| `pkg/config/config.go`              | Add `Cron`, `CronTimeout` to `PipelineTrigger`                                       |
| `pkg/pipeline/loader.go`            | Add `Cron`, `CronTimeout` to `Trigger`; validation; update `LoadConfig`              |
| `pkg/pipeline/engine.go`            | Embed cron scheduler; per-pipeline mutex map; `Stop()` with timeout; clock interface |
| `pkg/pipeline/clock.go`             | Clock interface + RealClock implementation                                           |
| `internal/server/pipeline.go`       | fx hook for `engine.Stop()`                                                          |
| `pkg/metrics/pipeline.go`           | Cron-specific metrics                                                                |
| `docs/reference/pipelines.yaml`     | Cron trigger example                                                                 |
| `pkg/pipeline/engine_test.go`       | Cron scheduling tests (with FakeClock)                                               |
| `pkg/pipeline/loader_test.go`       | LoadConfig cron mapping + validation tests                                           |
| `pkg/config/config_test.go`         | Config parse cron field tests                                                        |
| `tests/specs/pipeline_spec_test.go` | BDD cron trigger specs (with FakeClock)                                              |
