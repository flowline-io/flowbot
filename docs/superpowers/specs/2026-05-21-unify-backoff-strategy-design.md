# Unify Backoff Strategy

## Context

Three independent backoff/retry implementations share the `cenkalti/backoff` v2 library, but differ in loop logic, config types, and retry decisions:

| Location | Config Type | Retry Loop Lines | Unique Logic |
|---|---|---|---|
| `pkg/pipeline/engine.go` | `types.RetryConfig` | ~40 lines | `isRetryable()` + `RetryOn` code filtering |
| `pkg/workflow/workflow.go` | `types.RetryConfig` | ~70 lines (two functions) | `runWithRetry()` + `runEngineWithRetry()` |
| `pkg/event/middleware.go` | `event.Retry` (custom) | ~60 lines | `MaxElapsedTime` + `OnRetryHook` |

Pipeline and workflow share `types.RetryConfig.BuildBackOff()` for delay computation, but their retry loops are near-identical copy-paste. The event module is entirely separate and does not use `types.RetryConfig`.

## Motivation

- **Eliminate duplication**: Three retry loops with near-identical structure require synchronized changes across all three locations.
- **Unified interface**: `event.Retry` has richer features (`MaxElapsedTime`, `OnRetryHook`) that should be aligned with pipeline/workflow.
- **Standardized jitter**: Current jitter uses `RandomizationFactor: 0.5`. Replace with industry-standard full jitter (random in `[0, delay)`).
- **Adaptive backoff**: Add success-based adaptive backoff — halve delay after success, double after failure — to reduce recovery latency after extended idle periods.

## Design

### 1. New Package `pkg/backoff/`

Single API entry point. All consumers use one `Do` function for retryable operations.

```go
package backoff

type Config struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
    Jitter          bool
    Adaptive        bool
    RetryOn         []string
    IsRetryable     func(err error) bool
    MaxElapsedTime  time.Duration
    OnRetry         func(attempt int, delay time.Duration, err error)
}

// Do executes fn, retrying on error according to cfg.
// Returns total attempts and the last error (nil on success).
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) (attempt int, err error)

// Middleware wraps a Watermill handler with retry behavior.
func Middleware(cfg Config, logger watermill.LoggerAdapter) message.HandlerMiddleware
```

**Key design principles**:
- `Do` is the single retry loop implementation. Pipeline, workflow, and event no longer maintain their own loops.
- `Config` aggregates all existing differentiated fields (`RetryOn`, `MaxElapsedTime`, `OnRetry`).
- Zero-value semantics: `MaxAttempts=0` means no retry; `Multiplier=0` defaults to 2.0; `MaxElapsedTime=0` means no time limit.

### 2. Adaptive Backoff Algorithm

```
delay = InitialInterval
for attempt := 1; attempt <= MaxAttempts; attempt++ {
    err := fn(ctx)
    if err == nil {
        if Adaptive { delay = max(InitialInterval, delay / 2) }
        return attempt, nil
    }
    if !shouldRetry(attempt, err, cfg) { return attempt, err }
    sleep(jitter(delay))
    if Adaptive {
        delay = min(MaxInterval, delay * 2)
    } else {
        delay = min(MaxInterval, delay * Multiplier)
    }
    if cfg.MaxElapsedTime > 0 && elapsed >= MaxElapsedTime { return attempt, err }
}
```

- **Success halves delay**: After consecutive successes, delay gradually decreases, quickly returning to low-latency state.
- **Failure doubles delay**: Consistent with traditional exponential backoff.
- **Boundaries**: Delay is always clamped to `[InitialInterval, MaxInterval]`.

### 3. Full Jitter

Abandon `cenkalti/backoff`'s `RandomizationFactor` offset mode in favor of full jitter:

```go
func jitter(d time.Duration) time.Duration {
    return time.Duration(rand.Int64N(int64(d)))
}
```

Uniformly random in `[0, delay)`, which distributes retry traffic more evenly than half-range jitter.

### 4. Retry Decision (`shouldRetry`)

Migrated from `pkg/pipeline/engine.go:isRetryable()`:

1. `cfg.RetryOn` empty and `cfg.IsRetryable` nil → all errors are retryable.
2. `cfg.IsRetryable` non-nil → callback result takes precedence, overrides `RetryOn`.
3. `cfg.RetryOn` non-empty and `IsRetryable` nil → match against `types.Error.Code`; `*types.Error` with `Retryable: true` also matches.
4. None of the above → not retryable, return immediately.

### 5. Backward Compatibility: `types.RetryConfig.ToBackoffConfig()`

```go
func (rc *RetryConfig) ToBackoffConfig() backoff.Config
```

Maps existing `RetryConfig` fields to `backoff.Config`. Kept in `pkg/types/workflow.go` marked as deprecated for gradual migration. The pipeline loader's `convertRetryConfig()` is changed to directly construct `backoff.Config`.

## Migration Plan

| Module | Current Code | After Migration |
|---|---|---|
| `pkg/pipeline/engine.go:executeStep()` | Inline ~40 line retry loop | `backoff.Do(ctx, cfg, fn)` |
| `pkg/pipeline/engine.go:isRetryable()` | ~30 lines pipeline-private | Moved into `pkg/backoff/shouldRetry()` |
| `pkg/workflow/workflow.go:runWithRetry()` | ~35 lines | `backoff.Do(ctx, cfg, fn)` |
| `pkg/workflow/workflow.go:runEngineWithRetry()` | ~35 lines | Merged into above, separate function no longer needed |
| `pkg/event/middleware.go` | ~100 lines `event.Retry` + `Middleware()` | `backoff.Middleware(cfg, logger)` |
| `pkg/types/workflow.go` | `BuildBackOff()` + `RetryEnabled()` | Add `ToBackoffConfig()`, mark old methods deprecated |
| `pkg/event/pubsub.go` | `event.Retry{...}` construction | `backoff.Config{...}` construction |
| `pkg/pipeline/loader.go` | `convertRetryConfig()` returns `*types.RetryConfig` | Returns `backoff.Config` |

**Kept unchanged**:
- `pkg/types/errors.go` `Error.Retryable` field — `shouldRetry` continues to use it.
- `pkg/metrics/pipeline.go` / `pkg/metrics/workflow.go` `IncStepRetry()` counters — callers track after `Do` returns.
- All BDD and unit tests — update imports, behavior unchanged.

## Testing

### TDD Unit Tests (`pkg/backoff/backoff_test.go`)

Each category with >=3 cases, table-driven pattern:

1. **Basic retry**: success on first try / success after 2 failures / exceed MaxAttempts permanent failure
2. **RetryOn filtering**: matching code triggers retry / non-matching returns immediately / empty RetryOn retries all
3. **IsRetryable callback**: custom predicate overrides / returns false terminates immediately / returns true continues
4. **Full Jitter**: sleep within [0, delay) / average near delay/2 across calls / disabled jitter yields exact delay
5. **Adaptive**: delay halves after success / delay doubles after failure / clamped to [InitialInterval, MaxInterval]
6. **MaxElapsedTime**: last retry before timeout / timeout terminates immediately / zero value means unlimited
7. **Context cancellation**: ctx.Cancel returns immediately / ctx.Deadline expires / ctx runs normally
8. **OnRetry callback**: fires correctly / nil callback does not panic / receives correct attempt and error
9. **Middleware**: successful handler no retry / failing handler retries / permanent failure returns final error

### BDD Test Migration

- `tests/specs/pipeline_spec_test.go`: update imports, re-run to confirm.
- `tests/specs/workflow_spec_test.go`: same.
- New `tests/specs/backoff_spec_test.go`: end-to-end adaptive backoff scenario.

### Test Dependencies

- Unit tests have no external dependencies (use real `time.After` with short intervals, or a clock interface).
- BDD tests require Docker (`go tool task test:specs`).
