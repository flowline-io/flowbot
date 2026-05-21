# Ability Event Worker Pool

**Date**: 2026-05-21
**Status**: Draft
**Author**: Flowbot

## Problem

`ability.Invoke()` in `pkg/ability/invoke.go:131-140` spawns an unbounded goroutine per invocation for event emission (persisting `data_events` to PostgreSQL and publishing to Redis Stream). There is no concurrency control anywhere in the emission path. Under high load (e.g. 1000 concurrent HTTP requests), 1000 goroutines compete for database and Redis connections simultaneously, risking resource exhaustion.

## Solution

Replace the raw `go func()` with a goroutine pool backed by `github.com/panjf2000/ants/v2`. The pool is non-blocking with best-effort semantics: when the pool is full, new events are dropped (logged + metricked) rather than blocking the synchronous `Invoke` path.

### Scope

- Only the event emission goroutine (`invoke.go:131`). The synchronous invoker call remains unchanged.
- No changes to call sites (HTTP handlers, cron, pipeline, workflow) — those are already bounded by their respective concurrency models.
- No guaranteed-delivery outbox in this iteration; can be added later if needed.

### Components

| Component        | File                         | Purpose                                                       |
| ---------------- | ---------------------------- | ------------------------------------------------------------- |
| EventPool        | `pkg/ability/pool.go`        | `ants.Pool` wrapper: `Submit()`, `Shutdown()`, config parsing |
| Config           | `docs/reference/config.yaml` | New `ability.event_pool` section                              |
| Registry         | `pkg/ability/invoke.go`      | Replace `go func()` with `pool.Submit()`                      |
| AbilityCollector | `pkg/metrics/ability.go`     | New `event_dropped_total` counter                             |

### Architecture

```
ability.Invoke(ctx, capability, operation, params)
    |
    v
invoker(ctx, params)  ← synchronous, runs on caller goroutine
    |
    v
result.Events > 0 && emitter != nil?
    |-- no --> return result
    |
   yes
    |
    v
pool.Submit(func() { emitter(ctx, result) })
    |-- OK  --> return result (worker picks up task)
    |-- ErrPoolOverload --> log.Warn + metrics.inc(event_dropped) + return result
```

### Pool Configuration

| Parameter   | Key                                  | Default                                       | Description                   |
| ----------- | ------------------------------------ | --------------------------------------------- | ----------------------------- |
| Size        | `ability.event_pool.size`            | `0` (ants default: `runtime.GOMAXPROCS(-1)` ) | Max concurrent workers        |
| Expiry      | `ability.event_pool.expiry_duration` | `"30s"`                                       | Idle worker eviction interval |
| Nonblocking | —                                    | `true` (hardcoded)                            | Never block caller on submit  |

```yaml
ability:
  event_pool:
    size: 0
    expiry_duration: "30s"
```

### Error Handling

- **Pool full**: `ants.ErrPoolOverload` — log at WARN, increment `ability_event_dropped_total` counter, return normally from `Invoke`
- **Pool closed**: `ants.ErrPoolClosed` (during shutdown) — log at WARN, increment dropped counter
- **Worker panic**: `ants` recovers panics internally and logs to stderr; manual `defer recover()` in `invoke.go` removed (ants handles it)

### Graceful Shutdown

`ability.ShutdownPool()` calls `pool.ReleaseTimeout(30s)`, which:

1. Stops accepting new tasks
2. Waits up to 30s for in-flight tasks to complete
3. Forces termination on timeout

Called from `internal/server/server.go` in the `OnStop` hook alongside existing shutdown logic.

### Metrics

| Metric                        | Type    | Labels                              | Description                                   |
| ----------------------------- | ------- | ----------------------------------- | --------------------------------------------- |
| `ability_event_dropped_total` | Counter | `capability`, `operation`, `reason` | Events dropped due to pool overload or closed |

Added to `AbilityCollector` alongside existing `invoke_total`, `invoke_duration`, `invoke_error_total`.

### Testing Strategy

| Layer       | File                         | Cases                                                                                                                               |
| ----------- | ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Unit        | `pkg/ability/pool_test.go`   | Submit success, pool full returns error, shutdown prevents submits, worker panic recovered, concurrent submits ≤ pool size all pass |
| Unit        | `pkg/ability/invoke_test.go` | Invoke emits via pool (wait for async), pool full drops event (via mock), no events = no submit                                     |
| Integration | —                            | None; `ants` is already tested upstream; we test only our wrapper                                                                   |

Tests follow the existing TDD pattern: `t.Run(tt.name, t.Parallel())` inside a table of ≥3 cases.

### Dependencies

New dependency: `github.com/panjf2000/ants/v2`

### Anti-Patterns Avoided

- No change to `ability.Invoke` API signature
- No blocking in `Invoke` (nonblocking submit)
- No new goroutine spawn per invocation
- No cross-service logic in pool wrapper
- No `encoding/json` — config parsed via ent schema or Viper (existing patterns)
