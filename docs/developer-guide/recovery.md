# Recovery Manager

Restart recovery for incomplete pipeline runs and workflow jobs.

Source: `pkg/recovery/`

## Overview

When Flowbot restarts after a crash or intentional shutdown, long-running pipelines and workflows may be left in a non-terminal state (`PipelineStart`, `JobRunning`). The Recovery Manager scans for these incomplete executions and attempts to resume them.

```
Flowbot startup
    │
    ▼
Recovery Manager (pkg/recovery/recovery.go)
    │
    ├── Pipeline recovery
    │     ├── Query pipeline_runs WHERE status = PipelineStart
    │     ├── Check heartbeat freshness (stale threshold)
    │     ├── Skip active runs (recent heartbeat)
    │     ├── Mark expired runs as cancelled
    │     └── Resume stale runs (if auto_resume enabled)
    │
    └── Workflow recovery
          ├── Query jobs WHERE state = JobRunning
          ├── Mark stale jobs as failed
          └── Mark for resume (if auto_resume enabled)
```

## Configuration

Add a `recovery` section to `flowbot.yaml`:

```yaml
recovery:
  enabled: true # Enable the recovery manager
  stale_timeout: 5m # Time since last heartbeat before a run is considered stale
  auto_resume: true # Automatically resume stale runs (false = mark as cancelled/failed)
  max_resume_age: 24h # Maximum age of a run to attempt resumption (older runs are cancelled)
```

| Field            | Type     | Default | Description                                                                                                                                 |
| ---------------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `enabled`        | bool     | `false` | Master switch for recovery manager                                                                                                          |
| `stale_timeout`  | duration | `0s`    | Inactivity threshold; runs with no heartbeat for this duration are considered stale. `0` means no timeout (all incomplete runs are stale).  |
| `auto_resume`    | bool     | `false` | If `true`, stale runs are resumed. If `false`, stale runs are marked as `PipelineCancel` / `JobFailed`.                                     |
| `max_resume_age` | duration | `0s`    | Maximum allowed age of a run to attempt resumption. Runs older than this are cancelled regardless of `auto_resume`. `0` disables age check. |

## Pipeline Recovery

### How It Works

1. On startup, `Recover()` queries `pipeline_runs` where `status = PipelineStart`.
2. For each incomplete run:
   a. **Active check**: If `last_heartbeat` is recent (within `stale_timeout`), the run is still being executed — skip it.
   b. **Age check**: If `started_at` is older than `max_resume_age`, mark as `PipelineCancel` — too old to resume.
   c. **Auto-resume**: If `auto_resume: true`, the run is marked for resumption (actual resume is wired externally via `Engine.ResumePipeline`).
   d. **Manual recovery**: If `auto_resume: false`, the run is marked as `PipelineCancel` — administrator must manually initiate recovery.

### Staleness Detection

```
isStale(run):
  if stale_timeout <= 0: return true          # all runs immediately stale
  if last_heartbeat == nil:
    return time.Since(started_at) > stale_timeout  # fallback to started_at
  return time.Since(last_heartbeat) > stale_timeout
```

A run is considered "active" (not stale) if its `last_heartbeat` was updated within `stale_timeout`. This requires the pipeline to have `resumable: true` (which enables the heartbeat goroutine).

### Resume Process

When a pipeline is resumed:

1. `Engine.ResumePipeline(ctx, runID)` loads the run record to get the pipeline name.
2. The checkpoint data (`pipeline_runs.checkpoint_data`) is deserialized.
3. The pipeline definition is matched by name (must have `resumable: true`).
4. `RenderContext` is reconstructed from the saved step results.
5. Execution continues from the checkpointed `step_index`.
6. New checkpoints are saved before each remaining step.
7. On completion, the run is marked `PipelineDone` or `PipelineCancel`.

## Workflow Recovery

### How It Works

1. On startup, `Recover()` queries `jobs` where `state = JobRunning`.
2. For each incomplete job:
   a. If `auto_resume: false`, mark as `JobFailed` — administrator intervention required.
   b. If `auto_resume: true`, the job is marked for resumption.

Workflow jobs don't currently have a heartbeat mechanism. The recovery manager treats all incomplete jobs as immediately stale.

### Limitations

- Workflow recovery currently only identifies stale jobs for manual/external resumption.
- The actual resume execution from a specific step requires the workflow YAML and task definitions — these must still be available on disk.
- For workflows with `resumable: true`, step states are written to the `steps` table, but automatic resume from the last incomplete step is not yet implemented.

## Admin Endpoints

The Recovery Manager exposes methods for programmatic access:

```go
// List incomplete pipeline runs
manager.GetIncompletePipelines()  // → []*model.PipelineRun

// List incomplete workflow jobs
manager.GetIncompleteWorkflows()  // → []*model.Job
```

HTTP endpoints can be built on top of these methods (not yet implemented):

```
GET  /service/recovery/incomplete           # list all incomplete runs
POST /service/recovery/resume/pipeline/:id  # manually resume a pipeline
POST /service/recovery/resume/workflow/:id  # manually resume a workflow
```

## Best Practices

### Enable for Long-Running Workloads

```
resumable: true   →  pipelines that process batch RSS feeds, large file imports
resumable: false  →  fast pipelines (< 5 seconds), simple notifications
```

### Tune the Stale Timeout

```
stale_timeout: 2m   →  fast pipelines, aggressive recovery
stale_timeout: 10m  →  batch processing, lenient recovery
stale_timeout: 1h   →  very slow pipelines, conservative
```

### Safety

- Recovery only touches runs in `PipelineStart` status. Runs that already completed (`PipelineDone` / `PipelineCancel`) are never touched.
- The idempotency guard (`event_consumptions`) prevents a recovered run from re-processing an already-consumed event.
- The `max_resume_age` cap prevents attempting to resume runs that are so old the original event context is meaningless.

## Testing

```bash
go test ./pkg/recovery/...
go test ./pkg/pipeline/...     # ResumePipeline tests
go test ./internal/store/...   # PipelineStore checkpoint tests
```

## Dependencies

- Requires MySQL for state persistence (checkpoint data, heartbeat timestamps).
- Pipeline recovery requires pipeline definitions to still be present in the running config.
- Workflow recovery requires workflow YAML files to still exist on disk.
- The `resumable` flag must be set on the pipeline/workflow definition to opt in.
