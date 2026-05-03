# Pipeline Engine

Event-driven pipeline automation with retry, checkpointing, and restart recovery.

Source: `pkg/pipeline/`

## Overview

The pipeline engine executes multi-step workflows in response to `DataEvent` messages published via Redis Stream. Each pipeline is defined in YAML configuration and consists of a trigger event and an ordered sequence of steps that invoke capability operations.

```
DataEvent (MySQL data_events + Redis Stream)
    │
    ▼
Pipeline Engine (pkg/pipeline/engine.go)
    │
    ├── Idempotency check (event_consumptions)
    ├── Create pipeline_run record
    ├── For each step:
    │     ├── Save checkpoint (if resumable)
    │     ├── Render template params
    │     ├── Create step_run record
    │     ├── ability.Invoke (with retry)
    │     └── Update step_run result
    └── Update pipeline_run status
```

## YAML Schema

Pipelines are defined under the `pipelines` key in `flowbot.yaml`:

```yaml
pipelines:
  - name: rss_fetch_and_notify          # unique name, used as consumer_name
    description: "Fetch RSS feeds and send notification"
    enabled: true                        # false to skip loading
    resumable: true                      # enable checkpoint + restart recovery

    trigger:
      event: rss.fetch.requested        # DataEvent.EventType to match

    steps:
      - name: fetch_feeds
        capability: rss                  # capability type
        operation: fetch                 # operation name
        params:                          # template-rendered input
          url: "{{event.url}}"
          max_items: 10
        retry:                           # step-level retry (optional)
          max_attempts: 3
          delay: 1s
          backoff: exponential           # fixed | linear | exponential
          max_delay: 60s
          jitter: true
          retry_on:                      # filter which errors to retry
            - timeout
            - rate_limited

      - name: send_notification
        capability: notify
        operation: send
        params:
          channel: slack
          message: "New feeds: {{step "fetch_feeds" "count"}}"
```

## Retry Strategy

### Configuration

Each pipeline step can specify an optional `retry` block. If omitted, the step runs exactly once.

| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `max_attempts` | int | `0` | Maximum retry attempts. `0` disables retry. |
| `delay` | duration | `0s` | Initial delay before first retry |
| `backoff` | string | `"exponential"` | `fixed` (constant delay), `linear` (multiplier=1.0), `exponential` (multiplier=2.0) |
| `max_delay` | duration | `0s` | Caps the delay between retries |
| `jitter` | bool | `false` | Adds +/-50% randomization to delay |
| `retry_on` | []string | (all errors) | Filter: only retry errors matching these codes or with `Retryable=true` |

### Behavior

1. On first failure, the engine checks if `max_attempts > 0`.
2. If `retry_on` is set, the error is checked against the filter:
   - `types.Error.Retryable == true` always qualifies for retry.
   - `types.Error.Code` is matched against `retry_on` entries.
   - If no filter is configured, all errors are retried.
3. The engine waits for the computed delay, then retries.
4. Retries continue until success, `max_attempts` is exhausted, or context is cancelled.
5. Each attempt is recorded (the `attempt` column on `pipeline_step_runs`).

### Backoff Calculation

**Fixed** (`backoff: fixed`): every retry waits the same `delay`.  
**Linear** (`backoff: linear`): delay = `delay * attempt_number`, capped at `max_delay`.  
**Exponential** (`backoff: exponential`): delay = `delay * 2^(attempt-1)`, capped at `max_delay`.

Jitter is only applied to linear and exponential modes (built on `ExponentialBackOff`).

### Database Recording

The `pipeline_step_runs` table tracks retries:

| Column | Description |
| ------ | ----------- |
| `attempt` | Count of attempts including the first (1-based) |
| `retry_config` | JSON snapshot of the retry configuration used |

## Checkpointing

### Enabling

Set `resumable: true` on the pipeline definition. Without this flag, no checkpoints are saved.

### Save Mechanics

Before each step executes, the engine serializes and persists a `CheckpointData` JSON blob to `pipeline_runs.checkpoint_data`:

```json
{
  "step_index": 2,
  "step_results": {
    "fetch_feeds": {
      "name": "fetch_feeds",
      "capability": "rss",
      "operation": "fetch",
      "output": {"count": 42, "items": [...]},
      "completed_at": "2026-05-03T10:00:00Z"
    }
  },
  "event": { /* the triggering DataEvent */ },
  "heartbeat_at": "2026-05-03T10:00:05Z"
}
```

This captures enough state to reconstruct the `RenderContext` and resume from the checkpointed step.

### Heartbeat

When `resumable: true`, each step starts a background goroutine that writes `last_heartbeat` to `pipeline_runs` every 30 seconds. This allows the Recovery Manager to distinguish genuinely in-progress runs from orphaned ones.

Heartbeats stop automatically when the step completes (context cancelled via `defer hbCancel()`).

### Resume

`Engine.ResumePipeline(ctx, runID)` restores execution from the last checkpoint:
1. Loads `pipeline_runs` to get the pipeline name.
2. Loads `checkpoint_data` to get step index, step results, and the original event.
3. Matches the pipeline definition by name.
4. Reconstructs `RenderContext` from saved step results.
5. Continues executing from `step_index`, saving new checkpoints along the way.

## Execution States

| State | Value | Meaning |
| ----- | ----- | ------- |
| `PipelineStateUnknown` | 0 | Default |
| `PipelineStart` | 1 | Run in progress |
| `PipelineDone` | 2 | All steps succeeded |
| `PipelineCancel` | 3 | Step failed or run cancelled |

## Database Tables

| Table | Purpose |
| ----- | ------- |
| `pipeline_definitions` | Persisted YAML definitions (upserted on startup) |
| `pipeline_runs` | One row per pipeline execution: status, error, checkpoint, heartbeat |
| `pipeline_step_runs` | Per-step execution: params, result, attempt, status, error |
| `event_consumptions` | Idempotency guard: `(consumer_name, event_id)` unique |

## Event Flow

1. `ability.Invoke` returns `InvokeResult` with `Events` — a list of business events (`bookmark.created`, `rss.item.fetched`, etc.).
2. The event emitter (registered in `initPipeline`) creates a `DataEvent` and persists it:
   - `data_events` table (durable store)
   - `event_outbox` table (transactional outbox pattern)
3. The event is published to Redis Stream topic `pipeline:data_event`.
4. A Watermill consumer deserializes the event and calls `engine.Handler()`.
5. The engine runs `FindByEvent` to match pipeline definitions, executes them sequentially.

## Idempotency

Each pipeline run is gated by `event_consumptions` which has a unique composite index on `(consumer_name, event_id)`. Before execution:
1. `HasConsumed(pipelineName, eventID)` checks if this pipeline already processed this event.
2. If consumed, the event is skipped (logged, no error).
3. Otherwise, `RecordConsumption` inserts a row, then the pipeline executes.

This guarantees at-most-once processing per (pipeline, event) pair.

## Template Rendering

Step `params` are rendered through the template engine before each invocation. See [Pipeline Template Engine](pipeline-template.md) for syntax reference.

## Testing

```bash
go test ./pkg/pipeline/...        # Unit tests
go test ./pkg/pipeline/template/...  # Template engine tests
```

## Recovery

See [Recovery Manager](../developer-guide/recovery.md) for restart recovery of incomplete pipeline runs.
