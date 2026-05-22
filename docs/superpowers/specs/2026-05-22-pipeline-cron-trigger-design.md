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

`PipelineTrigger` gains a `Cron` field:

```go
type PipelineTrigger struct {
    Event string `json:"event" yaml:"event" mapstructure:"event"`
    Cron  string `json:"cron" yaml:"cron" mapstructure:"cron"`
}
```

`Event` and `Cron` can coexist. When both are set the pipeline triggers
from either source.

Example `pipelines.yaml`:

```yaml
- name: daily_cleanup
  description: "Daily cleanup job"
  enabled: false
  resumable: false
  trigger:
    cron: "0 3 * * *"
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
    Event string
    Cron  string
}
```

`LoadConfig` maps both fields.

### Cron expression format

Compatible with `github.com/flc1125/go-cron/v4` (already a project
dependency). Supports standard 5-field (`minute hour dom month dow`),
optional seconds, and descriptors (`@every 1h`, `@daily`).

### Engine-embedded scheduler (`pkg/pipeline/engine.go`)

- `Engine` holds `*cron.Cron` field.
- `NewEngine` creates the scheduler, registers cron-enabled definitions, starts.
- `Engine.Stop()` stops the scheduler and waits for in-flight jobs.

Each cron job runs independently:

1. Checks per-pipeline mutex to prevent overlapping runs.
2. Creates context with `auth.SystemCronContext()` and 10-minute timeout.
3. Synthesizes a `DataEvent`:
   - `EventID`: `cron:<pipeline-name>:<unix-millis>` (unique per tick)
   - `EventType`: `pipeline.cron:<pipeline-name>`
   - `Source`: `cron`
4. Calls `executePipeline(ctx, def, dataEvent)` — reuses the existing
   execution path including dedup, run records, checkpoints, metrics, and
   audit.

### Server lifecycle (`internal/server/pipeline.go`)

An fx lifecycle stop hook calls `engine.Stop()` for graceful shutdown
of the cron scheduler.

### Auth context

Cron-triggered pipeline steps use `auth.SystemCronContext()` so that
`ability.Invoke` calls are traceable to the cron subsystem.

## Key decisions

- **Mixed compatibility**: `event` and `cron` on the same trigger config
  are additive. A pipeline with both fires from either.
- **Synthetic DataEvent**: Cron does not emit events through Watermill / Redis
  Stream. It calls `executePipeline` directly with a fabricated event.
- **Concurrency guard**: Per-pipeline mutex (`sync.Mutex`) ensures a cron tick
  is skipped if the previous run is still executing.
- **Reuse execution path**: No new execution function. Cron shares the same
  `executePipeline` used by the event handler.

## Testing

### Unit tests (TDD, table-driven)

All test functions use `for _, tt := range tests { t.Run(tt.name, ...) }`
with at least 3 cases per table. Happy path first, error cases required.

**`pkg/config/config_test.go`** -- parse cron field from YAML:

- `tt.name`: "event only trigger", "cron only trigger", "both event and cron"
- verify `PipelineTrigger.Cron` is empty/non-empty as expected

**`pkg/pipeline/loader_test.go`** -- `LoadConfig` maps cron:

- `tt.name`: "event only definition", "cron only definition", "both triggers definition"
- verify `Trigger.Cron` and `Trigger.Event` mapped correctly

**`pkg/pipeline/engine_test.go`** -- cron engine behavior:

- `NewEngine` registers cron definitions: verify cron scheduler has expected number of entries (use a `CronEntryCount()` accessor or test via behavior)
- Concurrency guard: launch a long-running pipeline, fire the cron job manually, verify the second invocation is skipped
- `Stop()` clean shutdown: add a cron job, call `Stop()`, verify no more executions occur
- Synthetic event: verify `EventID` format `cron:<name>:<unix-millis>`, `EventType` format `pipeline.cron:<name>`, `Source` is `cron`

### BDD specs (Ginkgo v2 + Gomega)

`tests/specs/pipeline_spec_test.go` -- `Describe("Cron trigger")`:

- `It("executes pipeline on cron schedule")`: create pipeline with `@every 100ms` trigger, assert at least N runs complete within a timeout
- `It("skips overlapping runs")`: create pipeline with a blocking step, assert only one run is in-flight at a time
- `It("stops execution on engine shutdown")`: create pipeline with `@every 50ms`, call `Stop()`, assert no runs occur after stop
- `It("supports mixed event and cron trigger")`: pipeline with both `event` and `cron` fires from both sources
- `It("records correct DataEvent for cron run")`: assert run records have `event_type` = `pipeline.cron:<name>` and `source` = `cron`

## Files affected

| File | Change |
|------|--------|
| `pkg/config/config.go` | Add `Cron` to `PipelineTrigger` |
| `pkg/pipeline/loader.go` | Add `Cron` to `Trigger`; update `LoadConfig` |
| `pkg/pipeline/engine.go` | Embed cron scheduler; add `Stop()` |
| `internal/server/pipeline.go` | fx hook for `engine.Stop()` |
| `docs/reference/pipelines.yaml` | Cron trigger example |
| `pkg/pipeline/engine_test.go` | Cron scheduling tests |
| `pkg/pipeline/loader_test.go` | LoadConfig cron mapping tests |
| `pkg/config/config_test.go` | Config parse cron field tests |
| `tests/specs/pipeline_spec_test.go` | BDD cron trigger specs |
