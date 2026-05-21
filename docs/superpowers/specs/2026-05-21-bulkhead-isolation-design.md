# Bulkhead Isolation for Ability Invocations

**Date**: 2026-05-21
**Status**: Draft
**Author**: Flowbot

## Problem

`ability.Invoke()` in `pkg/ability/invoke.go:214` calls provider invokers synchronously on the caller's goroutine with no concurrency guard. A slow or stalled provider can consume all available goroutines, starving other providers and degrading the entire system. There is no per-capability isolation.

## Solution

Introduce a bulkhead pattern: a semaphore per capability with bounded concurrency and a bounded wait queue. When a capability's slots are full, callers queue with a timeout rather than consuming unbounded resources.

### Scope

- Only the `invoker(ctx, params)` call in `ability.Invoke()` is wrapped.
- Bulkhead instances are created lazily per capability.
- No configuration surface — defaults are hardcoded.
- No adaptive sizing in this iteration.

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Isolation granularity | Per capability | True bulkhead: a slow provider blocks only itself |
| Full behavior | Queue + timeout | Avoids immediate failure for transient spikes |
| Timeout source | Hardcoded 30s default | Simple, consistent, no config burden |
| Semaphore size | `GOMAXPROCS * 4` | Scales with machine, I/O-bound work benefits from oversubscription |
| Queue capacity | Same as semaphore (`GOMAXPROCS * 4`) | 2x total waiting capacity feels appropriate short of data |

## Architecture

```
ability.Invoke(ctx, capability="llm", operation="chat", params)
    |
    v
bulkhead.Get("llm").Do(ctx, func() error { return invoker(ctx, params) })
    |
    ├── 1. queue <- token    (non-blocking, full → ErrBulkheadFull)
    |
    ├── 2. select {
    |       case sem <- token:   proceed to execution
    |       case <-timeoutCh:    release queue slot → ErrBulkheadTimeout
    |       case <-ctx.Done():   release queue slot → ctx.Err()
    |     }
    |
    ├── 3. fn()              (execute on caller goroutine)
    |
    └── 4. <-sem; consume queue slot (release)
```

Each capability gets an independent `Bulkhead` instance with its own `sem` and `queue` channels. A slow `karakeep` call blocks only `karakeep` invocations; `llm` and `homelab` invocations are unaffected.

## API

```go
// Package bulkhead provides per-capability semaphore isolation for ability invocations.
package bulkhead

// Bulkhead controls concurrent access for a named resource.
type Bulkhead struct {
    name   string
    sem    chan struct{}
    queue  chan struct{}
    config config
}

// Do acquires a slot, waiting up to the configured timeout, then executes fn.
// Returns ErrBulkheadFull if the queue is at capacity.
// Returns ErrBulkheadTimeout if a slot is not acquired within the timeout.
// Returns ctx.Err() if the context is cancelled while waiting.
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error

// Get returns the Bulkhead for the given name, creating one with default config if needed.
func Get(name string) *Bulkhead
```

## Error Handling

```go
var (
    ErrBulkheadFull    = errors.New("bulkhead: queue full")
    ErrBulkheadTimeout = errors.New("bulkhead: wait timeout")
)
```

Mapping in `ability.Invoke()`:

| Bulkhead error | Returned error |
|----------------|----------------|
| `ErrBulkheadFull` | `types.Errorf(types.ErrRateLimited, ...)` |
| `ErrBulkheadTimeout` | `types.Errorf(types.ErrTimeout, ...)` |
| `ctx.Err()` | wrapped `ctx.Err()` |

Both errors are marked `Retryable: true` so `pkg/backoff/` retries them.

## Metrics

Added to `AbilityCollector` in `pkg/metrics/ability.go`:

| Method | Prometheus metric | Type |
|--------|-------------------|------|
| `IncBulkheadQueued` / `DecBulkheadQueued` | `ability_bulkhead_queued` | Gauge (capability label) |
| `IncBulkheadActive` / `DecBulkheadActive` | `ability_bulkhead_active` | Gauge (capability label) |
| `IncBulkheadDropped` | `ability_bulkhead_dropped_total` | Counter (capability, reason) |
| `ObserveBulkheadWaitDuration` | `ability_bulkhead_wait_seconds` | Histogram (capability) |

Logging: `flog.Warn` on dropped events only. Normal execution produces no log output.

## Files

| File | Change |
|------|--------|
| `pkg/bulkhead/bulkhead.go` | New: `Bulkhead` struct, `Do` method, sentinel errors |
| `pkg/bulkhead/bulkhead_test.go` | New: TDD unit tests |
| `pkg/bulkhead/manager.go` | New: global registry, `Get` function, default config |
| `pkg/metrics/ability.go` | Modify: add bulkhead gauge, counter, histogram |
| `pkg/ability/invoke.go` | Modify: wrap `invoker` call with `bulkhead.Get(...).Do(...)` |
| `pkg/types/errors.go` | No change needed — existing `ErrRateLimited` / `ErrTimeout` suffice |

## Testing

### Unit Tests (`pkg/bulkhead/bulkhead_test.go`)

Table-driven tests using `for _, tt := range tests { t.Run(tt.name, ...) }`:

1. **Acquire and release** — single goroutine acquires, executes, releases.
2. **Concurrent execution** — N goroutines acquire simultaneously, verify max N concurrent.
3. **Queue wait with timeout** — fill all slots, new caller waits and times out with `ErrBulkheadTimeout`.
4. **Queue full fast-fail** — fill slots + queue, new caller gets `ErrBulkheadFull`.
5. **Context cancellation** — waiting caller's ctx cancelled, returns `ctx.Err()`.
6. **-race clean** — no data races under concurrent use.
7. **Default size calculation** — verify `GOMAXPROCS() * 4`.
8. **Lazy singleton** — `Get("x")` twice returns same instance.

### BDD Tests (integration)

- Slow capability does not block a different capability.
- No goroutine leak under sustained load.
