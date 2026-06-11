# Health Dashboard Design

## Summary

Add a `/service/web/healthz` page providing a real-time system health dashboard,
auto-refreshing every 30s via HTMX polling. Covers database latency, runtime
metrics, capability health checks, and recent error log summary.

## Files

| Action   | File                                     | Purpose                                   |
| -------- | ---------------------------------------- | ----------------------------------------- |
| Modify   | `pkg/flog/flog.go`                       | Add 50-entry ring buffer for ERROR logs   |
| New      | `pkg/views/pages/healthz.templ`          | Full page wrapped in `@layout.Base`       |
| New      | `pkg/views/partials/healthz_status.templ`| Auto-refresh partial with all metrics     |
| Modify   | `internal/modules/web/healthz_webservice.go` | Add `healthzPage` handler + route rule    |
| Modify   | `internal/modules/web/module.go`         | Register rule in `Webservice()`           |
| Modify   | `pkg/views/layout/base.templ`            | Add "Health" nav link                     |

## Flog Ring Buffer

A thread-safe 50-entry ring buffer stores `{Time, Message, Caller}` for every
`flog.Err()` call. Exported via `flog.RecentErrors() []ErrorEntry`.

```go
type ErrorEntry struct {
    Time    time.Time `json:"time"`
    Message string    `json:"message"`
    Caller  string    `json:"caller,omitzero"`
}
```

## Route

```
GET /service/web/healthz
```

Registered in the web module's `Webservice()` ruleset with `route.WithNotAuth()`,
consistent with other web UI routes (cookie-based token validation, redirects to
`/service/web/login` on failure).

## Handler Logic

`healthzPage(ctx fiber.Ctx) error`:

1. Creates a context with 5s deadline.
2. Runs DB ping, Redis ping, and capability health checks concurrently.
3. Reads `runtime.ReadMemStats` and `flog.RecentErrors()` synchronously.
4. Any individual failure shows degraded status; never breaks the whole page.
5. Detects `HX-Request` header:
   - Present: renders `healthzStatus` partial only.
   - Absent: renders `healthzPage` full page with layout.

## Template Structure

### `healthz.templ` (page)

```
@layout.Base("Health — Flowbot") {
    header "System Health"
    @healthzStatus(data)
}
```

### `healthz_status.templ` (partial)

Wrapped in a `<div>` with `hx-get="/service/web/healthz" hx-trigger="every 30s" hx-swap="outerHTML"`.

Four DaisyUI card sections:

1. **DB Latency**
   - PostgreSQL ping duration (ms)
   - Redis ping duration (ms)
   - `stat` + `stat-value` components, green/yellow/red thresholds

2. **Runtime**
   - Goroutine count (`runtime.NumGoroutine`)
   - Heap alloc / total alloc / sys memory (`runtime.ReadMemStats`)
   - Last GC pause time (ns)
   - Four `stat` cards in a grid

3. **Capability Status**
   - Table: capability type, backend, app, status (healthy/unhealthy/error)
   - Uses `hub.Default.List()` to enumerate capabilities
   - Calls `ability.Invoke(ctx, desc.Type, "health")` per capability with 2s timeout
   - Capabilities without a health operation show "n/a"
   - Status badge: `badge-success` (healthy), `badge-error` (unhealthy/error)

4. **Recent Errors**
   - Scrollable list of last 10 entries from `flog.RecentErrors()`
   - Each entry shows timestamp + truncated message
   - Empty state: "No recent errors"

## Edge Cases

| Scenario                         | Behavior                                |
| -------------------------------- | --------------------------------------- |
| DB unreachable                   | Grey badge + "unreachable", -1ms        |
| Redis unreachable                | Grey badge + "unreachable", -1ms        |
| Capability health invoke timeout | "timeout" status, yellow badge          |
| Capability has no health op      | "n/a" status, grey badge                |
| No capabilities registered       | Empty table + "No capabilities" message |
| Error buffer empty               | "No recent errors"                      |
| HTMX poll request                | Return partial only, skip layout        |
| First page load                  | Full page with layout                   |

## Nav Link

Add `<a href="/service/web/healthz">Health</a>` to the navbar in
`pkg/views/layout/base.templ`, consistent with existing link style.

## Testing

- **TDD**: Unit tests for `flog.RecentErrors()` ring buffer behavior (fill,
  overflow, empty), handler HTMX header detection, data gathering with DB/Redis
  mocks.
- **BDD**: Spec verifying `/service/web/healthz` returns 200 with auth, 401
  without, and rendered page contains all four metric sections.
