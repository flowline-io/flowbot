# Workflow Concurrent Execution Design

**Date**: 2026-05-18
**Status**: Draft
**Scope**: `pkg/workflow/`, `pkg/types/workflow.go`, `pkg/pipeline/template/engine.go`

## Summary

Enable DAG-based parallel task execution in the workflow engine using the existing `Conn` dependency field (currently validation-only). A dependency-count scheduler with a configurable semaphore runs independent tasks concurrently. Pipelines remain sequential.

## Design Decisions

| Decision               | Choice                                                                       |
| ---------------------- | ---------------------------------------------------------------------------- |
| Scope                  | Workflow engine only; pipelines stay sequential                              |
| Dependencies           | Existing `WorkflowTask.Conn` field drives execution scheduling               |
| Error handling         | Fail-fast: cancel all running tasks on first error via `context.CancelFunc`  |
| Concurrency limit      | Per-workflow `max_concurrency` in YAML (default: 1 = sequential)             |
| Executor model         | Per-task `executor.Engine` instances, created on demand and closed after use |
| Checkpoint/Resume      | Per-task completion map (`CompletedTasks`) replaces linear `StepIndex`       |
| Scheduling             | Dependency-count decrement + ready queue + semaphore pool                    |
| Backward compatibility | `max_concurrency` absent or 1 falls through to existing sequential code path |

## Architecture

### New Component

**`pkg/workflow/scheduler.go`** — DAG-based parallel task scheduler containing:

- `buildDAG(tasks []WorkflowTask) *dag` — adjacency list + in-degree map
- `scheduler.Run(ctx, dag, taskMap, input, store, maxConcurrency, runID) (map[string]string, error)` — main execution loop

### Modified Files

| File                              | Change                                                                                                               |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `pkg/workflow/workflow.go`        | `Execute()` detects `max_concurrency > 1`, delegates to scheduler; sequential path unchanged                         |
| `pkg/workflow/loader.go`          | `ParseYAML()` reads `max_concurrency` from YAML; adds field to `ValidationResult`                                    |
| `pkg/workflow/persistence.go`     | `CheckpointData` gains `CompletedTasks map[string]bool`                                                              |
| `pkg/types/workflow.go`           | `WorkflowMetadata` gains `MaxConcurrency int`                                                                        |
| `pkg/executor/engine.go`          | No changes needed; existing `New(runtimeType)` factory already provides per-task engine instances                    |
| `pkg/pipeline/template/engine.go` | Move template `cache` from instance field to package-level `sync.Map` so per-task instances share compiled templates |

### Execution Flow

```
Runner.Execute()
  |
  +-- max_concurrency > 1 ?
  |     YES --> buildDAG(tasks) --> scheduler.Run()
  |     NO  --> existing sequential for-loop (unchanged)
  |
  scheduler.Run():
    1. ctx, cancel := context.WithCancel(parentCtx)
    2. sem := make(chan struct{}, maxConcurrency)
    3. ready := tasks with inDegree == 0
    4. results := map[string]string{} + sync.Mutex
    5. errOnce := sync.Once{} for fail-fast
    6.
        FOR ready queue not empty OR active > 0:
          WHILE ready not empty AND sem has capacity:
            id := pop(ready)
            sem <- struct{}{}
            wg.Add(1)
            go executeTask(id):
              defer wg.Done()
              defer func() { <-sem }()
              engine := executor.New(runtimeType)
              defer engine.Close()
              params, err := resolveParams(task.Params, results, input)
              if err != nil { cancel(); return }
              taskWithParams := applyParams(task, params)
              execTask, err := WorkflowTaskToTask(taskWithParams)
              if err != nil { cancel(); return }
              err = runWithRetry(ctx, engine, execTask, retryCfg, stepID, stepRun)
              mu.Lock()
              if err != nil:
                errOnce.Do(func() { cancel() })
              else:
                results[id] = result
                for each dependent dep of id:
                  dag[dep].inDegree--
                  if dag[dep].inDegree == 0:
                    ready = append(ready, dep)
                saveCheckpoint(CompletedTasks[id]=true)
              mu.Unlock()
        wg.Wait()
     7. return results, firstErr
```

## Data Model

### DAG Node

```go
type dagNode struct {
    task     types.WorkflowTask
    inDegree int            // count of unfinished dependencies
    deps     []string       // tasks that depend on this node (reverse edges)
}
```

### YAML Configuration

```yaml
name: save_and_track
max_concurrency: 3 # NEW: parallel task limit
resumable: true
pipeline:
  - fetch_data
  - archive_url
  - create_task
  - notify
tasks:
  - id: fetch_data
    action: capability:bookmark.list
    conn: [] # no deps, runs immediately
  - id: archive_url
    action: capability:archive.create
    conn: [fetch_data] # after fetch_data
  - id: create_task
    action: capability:kanban.create
    conn: [fetch_data] # parallel with archive_url
  - id: notify
    action: capability:notify.send
    conn: [archive_url, create_task] # after both
```

### CheckpointData

```go
type CheckpointData struct {
    StepIndex      int               // kept for sequential backward compat
    CompletedTasks map[string]bool   // NEW: task ID -> completed
    StepResults    map[string]string // keyed by task ID
    Input          types.KV
    HeartbeatAt    time.Time
}
```

### WorkflowMetadata

```go
type WorkflowMetadata struct {
    // ... existing fields ...
    MaxConcurrency int  // NEW: 0 or 1 = sequential, >1 = parallel DAG
}
```

## Error Handling

### Fail-Fast

- `sync.Once` wraps `context.CancelFunc` — only the first error triggers cancellation
- All goroutines check `ctx.Done()` at key points: template render, retry delay, executor start
- After `wg.Wait()`, the first error is returned to the caller

### Edge Cases

| Scenario                               | Behavior                                                                   |
| -------------------------------------- | -------------------------------------------------------------------------- |
| `Conn` references non-existent task ID | Rejected by `ValidateDAG()` before execution                               |
| Cycle in DAG                           | Rejected by `ValidateDAG()` before execution                               |
| Empty `Conn` on all tasks              | All tasks have `inDegree=0`, all run in parallel up to `max_concurrency`   |
| `max_concurrency=0` or absent          | Defaults to 1, uses sequential code path                                   |
| `max_concurrency > len(tasks)`         | Semaphore size = `max_concurrency`; no deadlock                            |
| Leaf node (no dependents)              | Completes without enqueuing anything; DAG completes when all done          |
| Template render fails                  | Treated as task failure, triggers fail-fast                                |
| Store unavailable                      | Store errors logged but not fatal (existing pattern); checkpoint not saved |
| Retry exhausted on one task            | Triggers fail-fast; other running tasks cancelled                          |

### Retry Integration

Each task retries independently via existing `runWithRetry`. Between retry attempts, `ctx.Done()` is checked. If another task failed and cancelled the context, the retrying task aborts immediately.

## Concurrency Safety

### Template Engine

The package-level `template.Engine` instance (`workflowEngine` in `workflow.go:605`) uses a `sync.Mutex` that serializes all rendering. For parallel execution, each goroutine creates its own `template.New()` instance. To preserve template compilation caching across instances, the `cache` field is moved from the `Engine` struct to a package-level `sync.Map` in `pkg/pipeline/template/engine.go`. Each instance uses the shared cache, and each has its own `mu` and `data` fields, so concurrent renders have no lock contention.

### Executor Engine

Each goroutine creates its own `executor.Engine` via `executor.New(runtimeType)`. The existing `Engine.mu` state machine is per-instance, so no lock contention. Each engine is closed via `defer engine.Close()` to release Docker/SSH resources.

### Results Map

Protected by `sync.Mutex`. Only the scheduler goroutine (via callbacks) and completing task goroutines access it. The lock is held briefly for map writes and checkpoint saves.

## Checkpoint & Resume

### Checkpoint During Execution

After each task completes successfully, the scheduler saves a checkpoint with its `CompletedTasks` map updated. The checkpoint captures the partial DAG state: which tasks are done and their results.

### Resume Algorithm

1. Load `CheckpointData` from store
2. Rebuild the same DAG from workflow definition
3. For each task in `CompletedTasks`: pre-mark as done, add results to `results` map
4. For remaining tasks: compute `inDegree` by subtracting completed dependencies
5. Tasks with `inDegree == 0` enter the ready queue
6. Resume execution from the partial state

A task whose dependencies are all in `CompletedTasks` but was itself not completed will be re-executed.

## Testing Strategy

### Unit Tests (TDD, table-driven)

- `TestBuildDAG` — valid DAG, cycle, missing reference, empty conn, single node
- `TestSchedulerRun` — happy path (2 parallel tasks), diamond DAG, fail-fast cancellation, empty tasks, max_concurrency enforcement
- `TestSchedulerResume` — resume from partial completion, resume with all done, resume with no checkpoints
- `TestParseYAMLMaxConcurrency` — valid values, zero, negative, missing
- `TestCheckpointDataCompletedTasks` — marshal/unmarshal round-trip

### BDD Specs (Ginkgo)

- Parallel execution of independent tasks
- Diamond DAG: A -> [B,C] -> D
- Fail-fast cancels sibling tasks
- max_concurrency limits active goroutines
- Resume after partial parallel execution
- Sequential fallback when max_concurrency=1
- Retry exhaust triggers fail-fast for parallel tasks

## Migration

No database migration required. `CheckpointData.CompletedTasks` is a new optional JSON field.

- **Sequential workflows** (`max_concurrency` absent or 1): Resume uses `StepIndex` as before. `CompletedTasks` is `nil` and ignored.
- **Parallel workflows** (`max_concurrency > 1`): Resume uses `CompletedTasks` to reconstruct DAG state. If `CompletedTasks` is `nil` (checkpoint from before this feature), resume falls through to sequential resume using `StepIndex`, which works correctly since parallel execution didn't exist yet.
- `ResumeWorkflow()` gains a branch: if `wf.MaxConcurrency > 1`, delegate to the parallel scheduler's resume path; otherwise use existing sequential resume.
