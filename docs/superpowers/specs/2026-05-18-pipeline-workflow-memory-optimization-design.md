# Pipeline + Workflow Memory Optimization

**Date**: 2026-05-18
**Scope**: `pkg/pipeline/`, `pkg/workflow/`, `pkg/pipeline/template/`
**Constraint**: Conservative — maintainability first, no public API changes

## Problem

Both pipeline and workflow engines allocate unnecessary memory on every execution:

1. **Template parsing per render** — `text/template.Parse` creates a full AST for each step's params string, even though param strings are static (from config). This is the dominant per-execution allocation.
2. **JSON buffer churn** — `sonic.Marshal` and `sonic.Unmarshal` allocate fresh `[]byte` buffers per call. Pipeline's `extractResult` round-trips through Marshal+Unmarshal for every non-map result.
3. **Executor runtime leak** — `workflow.Runner` creates 4 `executor.Engine` instances but never calls `Close()`, leaking Docker clients and SSH connections if the Runner is created per-request.

## Solution

Three independent, minimal changes:

### 1. Template Memoization Cache

**File**: `pkg/pipeline/template/engine.go`

Add a `sync.Map` cache to `Engine`, keyed by preprocessed template string, storing parsed `*txtpl.Template`. On `RenderString`, check cache before `txtpl.Parse`. If hit, skip parse and execute the cached template directly.

- Cache key: the output of `preprocessTemplate(tmpl)` — the canonical template source after `{{event.x}}` → `{{event "x"}}` conversion
- Cache value: `*txtpl.Template` with all funcMap functions registered
- Thread-safety: `sync.Map` is safe for concurrent reads (pipeline handlers are concurrent via Watermill router)
- Template functions (`step`, `event`, `input`) capture `*TemplateData` by reference — the cached template always sees current execution data via the template data map passed to `Execute`
- Workflow automatically benefits: its global `workflowEngine` already wraps `template.Engine`, so the cache is shared across all workflow executions

### 2. Sonic Internal Buffer Pooling

**Files**: `pkg/pipeline/engine.go`, `pkg/pipeline/template/engine.go`, `pkg/workflow/workflow.go`

Create a package-level frozen sonic config with internal pooling enabled:

```go
var pooledSonic = sonic.Config{PoolAlloc: true}.Froze()
```

Replace all bare `sonic.Marshal` and `sonic.Unmarshal` calls with the pooled variant. Affected call sites:

| Location                                   | Operation                         |
| ------------------------------------------ | --------------------------------- |
| `pkg/pipeline/engine.go:330`               | `extractResult` Marshal           |
| `pkg/pipeline/engine.go:335`               | `extractResult` Unmarshal         |
| `pkg/pipeline/template/engine.go:96`       | `json` template function Marshal  |
| `pkg/workflow/workflow.go:81`              | `marshalCapabilityParams` Marshal |
| `pkg/workflow/workflow.go:201`             | Run creation Marshal              |
| `pkg/workflow/workflow.go:261,307,407,449` | Result/step recording Marshal     |

sonic's `PoolAlloc` internally reuses encode/decode buffers per goroutine, which matches the sequential execution model of both engines.

### 3. Executor Lifecycle Cleanup

**File**: `pkg/workflow/workflow.go`

Add a `Close()` method to `Runner`:

```go
func (r *Runner) Close() error {
    for _, eng := range r.engines {
        if cerr := eng.Close(); cerr != nil {
            flog.Error(fmt.Errorf("[workflow] close engine: %w", cerr))
        }
    }
    return nil
}
```

Call `defer r.Close()` at the top of `Execute` and `ResumeWorkflow`. The underlying `executor.Engine.Close()` already exists at `pkg/executor/engine.go:55` and calls `runtime.Close()` on Docker/SSH/capability runtimes.

## Non-Goals

- No `sync.Pool` for `RenderContext`, `CheckpointData`, or other structs — adds complexity for small bounded allocations
- No pre-allocation of result maps — maps are allocate-once per execution, bounded by step count
- No shared template function map between packages — the functions are identical but sharing introduces cross-package coupling
- No mmap, arenas, or GC tuning — outside conservative scope

## Expected Impact

- **Template parsing**: O(steps × templates) → O(unique template strings). A 10-step pipeline with static params: 10 parses → 0 after cache warmup.
- **JSON buffers**: Reduced by sonic's internal goroutine-local buffer reuse, eliminating repeated `[]byte` allocs.
- **Executor cleanup**: No more leaked Docker clients or SSH connections per Runner lifecycle.

## Testing Strategy

- Existing unit tests in `pkg/pipeline/pipeline_test.go` and `pkg/workflow/workflow_test.go` must continue to pass
- Existing BDD specs cover pipeline and workflow execution paths — they catch regressions
- Template cache correctness: test that cached and uncached renders produce identical output
- `Runner.Close()`: test that calling `Close()` on an uninitialized engine (no runtime created) returns nil (idempotent)
- Both `go test ./pkg/pipeline/...` and `go test ./pkg/workflow/...` pass before and after changes

## Rollout

No feature flag needed — all changes are transparent to callers. Template cache is a pure performance optimization (same output, different path). Sonic pooling uses identical API surface. Executor cleanup is strictly additive (callers can ignore `Close()` and behavior degrades to current).
