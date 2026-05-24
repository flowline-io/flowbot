# Database Connection Pool Optimization: PostgreSQL Connection Management

Date: 2026-05-19
Status: design-approved

## Problem Statement

Audit of the PostgreSQL connection pool identified these issues:

1. **ConnMaxIdleTime hardcoded to 60s** — not configurable via `flowbot.yaml`, making it impossible to tune for production workloads without code changes.
2. **No connection health validation** — stale or dead connections (after network flaps, PostgreSQL restarts) are not detected until a query fails. No background health checking.
3. **No pool metrics exposure** — `store.Store.DbStats()` returns `sql.DBStats` but is not wired to any HTTP endpoint or Prometheus metric, leaving pool behavior invisible.
4. **Test path lacks pool config** — `internal/store/ent/ent.go` NewClient() calls `sql.Open()` with no pool tuning, meaning test connections have no limits.
5. **No timeout on initial Ping** — `adapter.Open()` calls `db.Ping()` without context timeout, so a slow database can hang startup indefinitely.
6. **No connection validation on borrow** — `database/sql` does not validate connections before lending them from the pool.

## Success Criterion

A PoolManager abstraction encapsulates pool configuration, health checking, and metrics instrumentation. All pool knobs are configurable via `flowbot.yaml`. Pool behavior is observable via Prometheus. The test path receives minimal safe defaults.

## Environment

- Go 1.26+
- PostgreSQL via `pgx/v5/stdlib` driver with `database/sql`
- Ent ORM via `entsql.OpenDB()`
- Prometheus metrics via existing `/metrics` endpoint
- Existing infra: `internal/store/postgres/adapter.go`, `internal/server/database.go`, `flowbot.yaml`

---

## Architecture

### Package Layout

```
internal/store/postgres/
  ├── adapter.go           # Existing adapter (modified: delegates pool setup to PoolManager)
  ├── adapter_test.go      # Existing tests (preserved)
  ├── pool.go              # NEW: PoolConfig, PoolManager, ApplyDefaults
  └── pool_test.go         # NEW: PoolManager unit tests
```

### Design Principles

- `PoolManager` wraps `*sql.DB` and manages pool lifecycle: config application, health pinging, metrics collection
- `adapter.Open()` creates a PoolManager after `sql.Open()`, delegates pool setup
- `PoolManager.Start(ctx)` / `PoolManager.Stop()` manage the background pinger goroutine
- `PoolManager.ApplyDefaults(db)` is a static function for the test path
- Prometheus metrics registered idempotently via `sync.Once`, updated each health check tick
- No breaking changes to `store.Adapter` interface or `store.Database` global pattern
- fx lifecycle hooks call `PoolManager.Stop()` on shutdown

---

## Components

### PoolConfig (pool.go)

```go
type PoolConfig struct {
    MaxOpenConns          int `json:"max_open_conns,omitempty"`
    MaxIdleConns          int `json:"max_idle_conns,omitempty"`
    ConnMaxLifetime       int `json:"conn_max_lifetime,omitempty"`
    ConnMaxIdleTime       int `json:"conn_max_idle_time,omitempty"`
    HealthCheckInterval   int `json:"pool_health_check_interval,omitempty"`
    HealthCheckTimeout    int `json:"pool_health_check_timeout,omitempty"`
}
```

Defaults (applied when value <= 0):
| Field | Default | Description |
|---|---|---|
| MaxOpenConns | 25 | Maximum open connections |
| MaxIdleConns | 5 | Maximum idle connections |
| ConnMaxLifetime | 300 | Max connection lifetime in seconds |
| ConnMaxIdleTime | 60 | Max idle time in seconds |
| HealthCheckInterval | 30 | Background pinger tick interval in seconds (0 = disabled) |
| HealthCheckTimeout | 5 | Per-ping context timeout in seconds |

### PoolManager (pool.go)

```go
type PoolManager struct {
    db     *sql.DB
    config PoolConfig
    cancel context.CancelFunc
    done   chan struct{}

    metricsOnce sync.Once
    // Prometheus metric collectors stored as fields
}
```

Methods:

- `ApplyConfig(db *sql.DB, cfg PoolConfig)` — applies all `SetMax*` on the `*sql.DB`
- `ApplyDefaults(db *sql.DB)` — static, applies conservative defaults (MaxOpen=10, MaxIdle=2, Lifetime=120s, IdleTime=30s)
- `Start(ctx context.Context)` — starts background pinger goroutine, registers metrics, returns immediately
- `Stop()` — cancels pinger context, waits for goroutine to finish (5s grace), safe to call on nil/stopped pools
- `healthCheck(ctx context.Context)` — calls `db.PingContext(ctx)`, logs warning on failure, increments error counter
- `registerMetrics()` — idempotent Prometheus registration via `sync.Once`
- `collectStats()` — reads `db.Stats()`, updates all gauges

### adapter.go Modifications

The `configType` struct gains three new fields:

```go
type configType struct {
    DSN                  string `json:"dsn,omitempty"`
    MaxOpenConns         int    `json:"max_open_conns,omitempty"`
    MaxIdleConns         int    `json:"max_idle_conns,omitempty"`
    ConnMaxLifetime      int    `json:"conn_max_lifetime,omitempty"`
    ConnMaxIdleTime      int    `json:"conn_max_idle_time,omitempty"`      // NEW
    SqlTimeout           int    `json:"sql_timeout,omitempty"`
    HealthCheckInterval  int    `json:"pool_health_check_interval,omitempty"` // NEW
    HealthCheckTimeout   int    `json:"pool_health_check_timeout,omitempty"`  // NEW
}
```

The `adapter` struct gains a `poolMgr *PoolManager` field.

`Open()` is modified:

1. `sql.Open("pgx", conf.DSN)` as before
2. Build `PoolConfig` from `configType`
3. `poolMgr := &PoolManager{}; poolMgr.ApplyConfig(db, poolCfg)` — replaces inline `SetMax*` calls
4. `poolMgr.Start(ctx)` — starts health pinger (non-blocking, failure logged not fatal)
5. Store `poolMgr` on adapter for shutdown

The `Close()` method (used by fx shutdown hook) calls `a.poolMgr.Stop()` before `a.db.Close()`.

The initial `db.Ping()` at startup now uses a configurable timeout (SqlTimeout) via `db.PingContext(ctx)`.

### ent.go Test Path

```go
func NewClient(dsn string) (*ent.Client, error) {
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, err
    }
    PoolManager.ApplyDefaults(db) // NEW: conservative pool limits
    drv := entsql.OpenDB("postgres", db)
    return ent.NewClient(ent.Driver(drv)), nil
}
```

---

## Data Flow

```
flowbot.yaml
  -> config.App.Store
  -> store.Store.Open(config.App.Store)
    -> adapter.Open(jsonConfig)
      -> configType populated from JSON (new fields included)
      -> sql.Open("pgx", dsn)
      -> poolMgr.ApplyConfig(db, PoolConfig{...})
          -> db.SetMaxOpenConns(25)
          -> db.SetMaxIdleConns(12)
          -> db.SetConnMaxLifetime(300s)
          -> db.SetConnMaxIdleTime(60s)
      -> db.PingContext(ctx) [with timeout]
      -> entsql.OpenDB("postgres", db) -> ent.NewClient()
      -> poolMgr.Start(ctx)
          -> registerMetrics() [sync.Once]
          -> go func() { ticker loop }()
      -> return adapter{db, client, poolMgr}

fx shutdown hook
  -> adapter.Close()
    -> poolMgr.Stop()
      -> cancel()
      -> <-done (with 5s timeout)
    -> db.Close()
```

Health pinger goroutine:

```
for {
    select {
    case <-ctx.Done():
        close(done); return
    case <-ticker.C:
        healthCheck(ctx) -> db.PingContext(healthCtx)
        collectStats()  -> update Prometheus gauges
    }
}
```

---

## Prometheus Metrics

All prefixed `flowbot_db_pool_`, registered via `prometheus.MustRegister`:

| Metric                                        | Type    | Labels | Description                               |
| --------------------------------------------- | ------- | ------ | ----------------------------------------- |
| `flowbot_db_pool_connections_open`            | Gauge   | —      | Current open connections                  |
| `flowbot_db_pool_connections_idle`            | Gauge   | —      | Current idle connections                  |
| `flowbot_db_pool_connections_in_use`          | Gauge   | —      | In-use connections                        |
| `flowbot_db_pool_wait_count_total`            | Counter | —      | Total connections waited for              |
| `flowbot_db_pool_wait_duration_seconds_total` | Counter | —      | Cumulative wait time in seconds           |
| `flowbot_db_pool_max_idle_closed_total`       | Counter | —      | Connections closed due to ConnMaxIdleTime |
| `flowbot_db_pool_max_lifetime_closed_total`   | Counter | —      | Connections closed due to ConnMaxLifetime |
| `flowbot_db_pool_health_check_total`          | Counter | —      | Total health pings performed              |
| `flowbot_db_pool_health_check_errors_total`   | Counter | —      | Failed health pings                       |

All counters reset on restart (standard Prometheus behavior). Gauges updated in `collectStats()` each health check tick alongside `db.Stats()`. Registration is idempotent via `sync.Once`.

---

## Error Handling

### PoolManager.Start()

| Scenario                               | Behavior                                                                              |
| -------------------------------------- | ------------------------------------------------------------------------------------- |
| `db.PingContext()` fails on first tick | Logs warning, increments error counter, does not crash                                |
| Ticker goroutine panics                | Recovered, logged, goroutine restarted after next tick                                |
| Prometheus metric already registered   | `sync.Once` prevents double registration, `AlreadyRegisteredError` caught and ignored |
| `HealthCheckInterval` is 0             | Pinger disabled, no goroutine started, `Start()` returns immediately                  |

### PoolManager.Stop()

| Scenario                          | Behavior                                |
| --------------------------------- | --------------------------------------- |
| Called on never-started pool      | No-op (nil cancel func)                 |
| Called twice                      | Safe (nil out cancel after first call)  |
| Goroutine does not exit within 5s | Timeout logged, `Stop()` returns anyway |

### adapter.Open()

| Scenario                                 | Behavior                                                   |
| ---------------------------------------- | ---------------------------------------------------------- |
| `sql.Open()` failure                     | Wrapped with `%w`, returned — no connection attempted      |
| `db.PingContext()` failure on startup    | Wrapped, returned as error — caller decides retry strategy |
| `entsql.OpenDB()` failure                | Wrapped, returned                                          |
| `PoolManager.ApplyConfig()`              | Never fails (all `Set*` methods are infallible)            |
| `PoolManager.Start()` initial ping fails | Logged as warning, startup continues                       |

### Runtime

| Scenario                          | Behavior                                                            |
| --------------------------------- | ------------------------------------------------------------------- |
| DB unreachable during health ping | Logged warning, error counter incremented, app continues            |
| Connection pool exhausted         | `database/sql` blocks/busy-waits, `pool_wait_count` metric captures |
| DB permanently down               | Pinger logs errors, app returns 503 from handlers, no crash         |

---

## Configuration

### flowbot.yaml Changes

Under `store_config.adapters.postgres`:

```yaml
store_config:
  use_adapter: "postgres"
  max_results: 1024
  adapters:
    postgres:
      dsn: "postgres://app:pass@localhost:5432/flowbot?sslmode=disable"
      max_open_conns: 25
      max_idle_conns: 12
      conn_max_lifetime: 300
      conn_max_idle_time: 60 # NEW: was hardcoded
      sql_timeout: 15
      pool_health_check_interval: 30 # NEW
      pool_health_check_timeout: 5 # NEW
    mysql:
      max_open_conns: 64
      max_idle_conns: 64
      conn_max_lifetime: 60
```

### docs/reference/config.yaml

Both updated to reflect the three new keys with documented defaults. The duplicate `conn_max_lifetime` entry in `docs/reference/config.yaml` is cleaned up.

---

## Testing Strategy

### Unit Tests (pool_test.go, TDD table-driven)

| Test                                  | What it verifies                                    |
| ------------------------------------- | --------------------------------------------------- |
| `TestPoolConfig_Defaults`             | Zero/negative values resolve to correct defaults    |
| `TestPoolConfig_FromJSON`             | JSON unmarshal populates all fields correctly       |
| `TestPoolManager_ApplyConfig`         | SetMax\* called with correct values                 |
| `TestPoolManager_ApplyDefaults`       | Sets expected conservative values                   |
| `TestPoolManager_StartPingerDisabled` | Interval=0 does not start goroutine                 |
| `TestPoolManager_StartStop`           | Start creates goroutine, Stop terminates it cleanly |
| `TestPoolManager_StopTwice`           | Double stop is safe                                 |
| `TestPoolManager_StopUnstarted`       | Stop on nil/never-started pool is safe              |
| `TestPoolManager_HealthCheckSuccess`  | Successful ping does not increment error counter    |
| `TestPoolManager_HealthCheckFailure`  | Failed ping increments error counter                |
| `TestPoolManager_CollectStats`        | db.Stats() fields map to correct metric values      |
| `TestPoolManager_MetricsIdempotent`   | Double registerMetrics does not panic               |

Each test uses `for _, tt := range tests { t.Run(tt.name, ...) }` pattern. Minimum 3 cases per table. Happy path first, error cases required.

### Integration Tests (require Docker PostgreSQL)

- `adapter.Open()` with full pool config creates working pool
- Pinger does not crash app when DB is killed and restarted
- Prometheus `/metrics` includes `flowbot_db_pool_*` metrics
- Config file changes propagate to runtime pool settings

### BDD Specs (Ginkgo v2 + Gomega)

- `pool_management_suite_test.go` with `SynchronizedBeforeSuite` + `GinkgoParallelProcess()` for per-process DB isolation
- Scenario: "healthy database pool starts and exposes metrics"
- Scenario: "pool survives transient database outage"
- Scenario: "pool metrics reflect connection exhaustion"

### Existing Tests

All existing tests must continue to pass. `adapter.Open()` signature does not change. `tests/specs/lifecycle.go` works with new `ApplyDefaults` via `ent.go`. Existing `adapter_test.go` preserved.

### Non-Breaking Guarantees

- `store.Adapter` interface unchanged
- `store.Database` global pattern unchanged
- `store.Store.Open()` signature unchanged
- `config.StoreType` struct unchanged (uses `map[string]any` already)
- New fields in `configType` use `omitempty` — missing keys in old config files use defaults
