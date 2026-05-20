# Redis Cache Strategy: Unified Abstraction Layer

Date: 2026-05-19
Status: design-approved

## Problem Statement

Audit of current Redis/cache usage identified five issues:

1. **Unbounded key growth** — cron filter sets, bloom filters, metrics keys, and some crawler keys have no TTL, growing without bound
2. **No cache hit/miss metrics** — Ristretto and Redis operations have zero application-level observability (only OTel command traces)
3. **Ad-hoc TTL policies** — TTL values are scattered as magic numbers (2min, 24h, etc.) with no consistency or documentation
4. **Underutilized in-process cache** — Ristretto only used once (alarm dedup), `sync.Map` template cache has no eviction, no tiered caching strategy
5. **No penetration protection** — no null-value caching to guard against cache miss storms

## Success Criterion

Code normalization: unified cache abstraction layer with standardized key naming, TTL policies, and metrics instrumentation. All cache operations go through well-defined interfaces.

## Environment

- Redis 8.0 (native Bloom filter via One Redis, no external modules required)
- Go 1.26+
- Existing infra: `pkg/cache/` (Ristretto), `pkg/rdb/` (Redis client), `pkg/stats/` (Prometheus pushgateway)

---

## Architecture

### Package Layout

```
pkg/cache/
  ├── cache.go           # RistrettoStore factory + global Instance (preserved for compat)
  ├── redis.go           # RedisStore — wraps rdb.Client, implements all interfaces
  ├── key.go             # Key builder enforcing {prefix}:{entity}:{identifier} naming
  ├── ttl.go             # TTL policy constants
  ├── null.go            # NullMarker for penetration protection
  ├── metrics.go         # Prometheus hit/miss/eviction/size instrumentation
  ├── types.go           # StringCache, ObjectCache[T], IntCache, SetCache, ListCache interfaces
  ├── cache_test.go      # Existing tests (preserved)
  ├── redis_test.go
  ├── key_test.go
  ├── null_test.go
  └── metrics_test.go
```

### Design Principles

- `RistrettoStore` handles in-process KV (enhanced version of current `cache.Instance`)
- `RedisStore` handles all Redis operations (replaces direct `rdb.Client` calls for caching)
- Each implements one or more interfaces from `types.go`
- All operations auto-record Prometheus metrics
- `rdb.Client` remains a package-level global but is no longer called directly by external packages for cache operations
- Bloom filter logic stays with `BFReserve/BFAdd` (Redis 8.0 native), add TTL only

### Dependency Injection

New provider in `internal/server/fx.go`:

```go
fx.Provide(
    cache.NewCache,           // existing RistrettoStore
    cache.NewRedisStore,      // new RedisStore(rdb.Client)
)
```

Modules inject `*cache.RedisStore` or specific interfaces instead of referencing `rdb.Client` directly.
`cache.NewCache` signature preserved for backward compatibility.

---

## Interfaces

```go
// StringCache covers raw string KV operations (chat sessions, online status, metrics).
type StringCache interface {
    Get(ctx context.Context, key Key) (string, bool, error)
    Set(ctx context.Context, key Key, value string, ttl TTL) error
    SetNX(ctx context.Context, key Key, value string, ttl TTL) (bool, error)
    Del(ctx context.Context, key Key) error
    Exists(ctx context.Context, key Key) (bool, error)
    Expire(ctx context.Context, key Key, ttl TTL) error
}

// ObjectCache covers typed KV for serializable values. Both Ristretto and Redis backends.
type ObjectCache[T any] interface {
    Get(ctx context.Context, key Key) (T, bool, error)
    Set(ctx context.Context, key Key, value T, ttl TTL) error
    Del(ctx context.Context, key Key) error
    Exists(ctx context.Context, key Key) (bool, error)
    // GetOrLoad is cache-aside with null-value penetration protection.
    GetOrLoad(ctx context.Context, key Key, ttl TTL, loader func(context.Context) (T, error)) (T, error)
}

// IntCache covers integer counters (Redis only).
type IntCache interface {
    Get(ctx context.Context, key Key) (int64, error)
    Set(ctx context.Context, key Key, value int64, ttl TTL) error
    Incr(ctx context.Context, key Key) (int64, error)
    IncrWithTTL(ctx context.Context, key Key, ttl TTL) (int64, error)
}

// SetCache covers set-based deduplication (Redis only).
type SetCache interface {
    Add(ctx context.Context, key Key, ttl TTL, members ...string) (int64, error)
    IsMember(ctx context.Context, key Key, member string) (bool, error)
    Members(ctx context.Context, key Key) ([]string, error)
    Remove(ctx context.Context, key Key, members ...string) (int64, error)
    Clear(ctx context.Context, key Key) error
}

// ListCache covers list-based aggregation buffers (Redis only).
type ListCache interface {
    Push(ctx context.Context, key Key, values ...string) (int64, error)
    Range(ctx context.Context, key Key, start, stop int64) ([]string, error)
    Len(ctx context.Context, key Key) (int64, error)
    Clear(ctx context.Context, key Key) error
}
```

Backend assignments:

- `RistrettoStore`: `StringCache`, `ObjectCache[T]`
- `RedisStore`: `StringCache`, `ObjectCache[T]`, `IntCache`, `SetCache`, `ListCache`

---

## Key Naming Convention

### Key Type

```go
type Key struct {
    Prefix     string   // Business domain: "online", "crawler", "cron", "notify", "chat"
    Entity     string   // Data purpose: "agent", "sent", "filter", "throttle", "session"
    Identifier string   // Business key, may contain additional ':' segments
}

func (k Key) String() string {
    return fmt.Sprintf("%s:%s:%s", k.Prefix, k.Entity, k.Identifier)
}
```

### Standardized Key Mapping

| Current Pattern                    | Standardized                        | Interface               |
| ---------------------------------- | ----------------------------------- | ----------------------- |
| `chat:<userId>`                    | `chat:session:<userId>`             | StringCache             |
| `online:<hostId>`                  | `online:agent:<hostId>`             | StringCache             |
| `online:<userId>`                  | `online:user:<userId>`              | StringCache             |
| `server:cron:online_count_last:*`  | `online:cron_count:<userId>`        | IntCache                |
| `crawler:<name>:sent`              | `crawler:sent:<name>`               | SetCache                |
| `crawler:<name>:todo`              | `crawler:todo:<name>`               | SetCache                |
| `crawler:<name>:sendtime`          | `crawler:sendtime:<name>`           | StringCache             |
| `cron:<name>:<uid>:filter`         | `cron:filter:<name>:<uid>`          | SetCache                |
| `bloom:unique:<id>`                | `cache:dedup:<id>`                  | SetCache (BF)           |
| `metrics:<metricName>`             | `metrics:gauge:<metricName>`        | IntCache                |
| `notify:throttle:<rid>:<et>:<ch>`  | `notify:throttle:<rid>:<et>:<ch>`   | IntCache                |
| `notify:agg:<rid>:<et>:<ch>`       | `notify:agg:buffer:<rid>:<et>:<ch>` | ListCache               |
| `notify:agg:timer:<rid>:<et>:<ch>` | `notify:agg:timer:<rid>:<et>:<ch>`  | StringCache (SETNX)     |
| `alarm:<sha1>`                     | `alarm:dedup:<sha1>` (in-process)   | ObjectCache (Ristretto) |

Rules:

- Prefix identifies business domain
- Entity identifies data purpose
- Identifier is the business primary key
- Shared/generic purpose keys use `cache:` prefix

---

## TTL Strategy

### Constant Definitions

```go
type TTL time.Duration

const (
    TTLNone     TTL = 0
    TTLMinute   TTL = TTL(time.Minute)
    TTLShort    TTL = TTL(2 * time.Minute)
    TTLMedium   TTL = TTL(10 * time.Minute)
    TTLLong     TTL = TTL(1 * time.Hour)
    TTLSession  TTL = TTL(24 * time.Hour)
    TTLDay      TTL = TTL(24 * time.Hour)
    TTLWeek     TTL = TTL(7 * 24 * time.Hour)
    TTLMonth    TTL = TTL(30 * 24 * time.Hour)
)
```

### Per-Key Allocation

| Key Pattern                 | TTL          | Rationale                        |
| --------------------------- | ------------ | -------------------------------- |
| `online:agent:*`            | `TTLShort`   | Fast heartbeat detection         |
| `online:user:*`             | `TTLSession` | User presence, longer window     |
| `online:cron_count:*`       | `TTLMedium`  | Transient cron state             |
| `chat:session:*`            | `TTLSession` | Session lifetime                 |
| `crawler:sent:*`            | `TTLMonth`   | Long-term dedup, auto-cleanup    |
| `crawler:todo:*`            | `TTLDay`     | Daily batch, fallback expiration |
| `crawler:sendtime:*`        | `TTLMedium`  | Track last send window           |
| `cron:filter:*`             | `TTLMonth`   | Prevents unbounded growth        |
| `metrics:gauge:*`           | `TTLMonth`   | Cron overwrites periodically     |
| `cache:dedup:*` (bloom)     | `TTLMonth`   | Prevents unbounded growth        |
| `notify:throttle:*`         | Dynamic      | Per-rule configurable window     |
| `notify:agg:buffer:*`       | Flush-only   | Short-lived, cleared on flush    |
| `notify:agg:timer:*`        | Dynamic      | Per-rule configurable window     |
| `alarm:dedup:*` (ristretto) | `TTLDay`     | In-process alarm dedup           |
| NullMarker                  | `TTLShort`   | Fast eviction for absent keys    |

### Configurable Overrides

```yaml
cache:
  ttl_overrides:
    crawler_sent: 720h
    cron_filter: 168h
```

---

## Metrics Instrumentation

Four Prometheus metrics following `pkg/stats/` conventions (counter + gauge, pushgateway push model).

### Metric Definitions

| Name                   | Type    | Labels    | Description                               |
| ---------------------- | ------- | --------- | ----------------------------------------- |
| `cache_hit_total`      | Counter | `backend` | Cache hits per backend (ristretto/redis)  |
| `cache_miss_total`     | Counter | `backend` | Cache misses per backend                  |
| `cache_eviction_total` | Counter | `backend` | Items evicted (explicit Del + TTL expiry) |
| `cache_size_bytes`     | Gauge   | `backend` | Approximate memory usage per backend      |

Label `backend` values: `ristretto`, `redis`.
No per-key labels to avoid high cardinality.

### Instrumentation Points

**RistrettoStore:**

- `Get()`: hit → `CacheHitTotal(ristretto).Inc()`, miss → `CacheMissTotal(ristretto).Inc()`
- `Set()` with eviction: increment `CacheEvictionTotal` when cost-based eviction occurs
- `size_bytes`: periodic sampling via Ristretto `Metrics.CostAdded() - Metrics.CostEvicted()`

**RedisStore:**

- `Get()`: value returned → hit, `redis.Nil` → miss
- `Del()`: increment `CacheEvictionTotal(redis)` for explicit deletes (TTL-based expiry not counted)
- `size_bytes`: periodic `DBSIZE` sampling

Periodic gauge updates run in background goroutine every 30 seconds.

---

## Penetration Protection

### NullMarker

```go
const NullMarker = "__cache_null__"
```

### GetOrLoad Pattern

`ObjectCache[T].GetOrLoad()` implements cache-aside with built-in null-value protection.
Since `T` may be a value type (struct, int, etc.), the loader must signal "not found" by returning `types.ErrNotFound`:

1. Check cache for `key`
2. If hit and value is `NullMarker`, return `types.ErrNotFound`
3. If hit and value is not `NullMarker`, deserialize and return the typed value
4. If miss, call `loader(ctx)` which returns `(T, error)`
5. If loader returns `types.ErrNotFound`, write `NullMarker` with `TTLShort` (2min) and return the error
6. If loader returns a non-nil error (not `ErrNotFound`), propagate the error without caching
7. If loader returns a value with nil error, write the serialized value with the requested TTL and return it

This is a composable building block; callers that don't need null-value protection can use plain `Get`/`Set`.

---

## Bloom Filter Retention

Redis 8.0 natively supports `BF.RESERVE`, `BF.ADD`, `BF.EXISTS` via One Redis.
No replacement needed. Changes from current code:

1. Move bloom operations from `pkg/rdb/unique.go` into `RedisStore` as a `BloomFilter` helper type
2. On `BFReserve`, follow with `EXPIRE <key> <TTLMonth>` to set 30-day TTL
3. Keep `BFAdd` atomic check-and-add semantics unchanged

---

## Migration Plan

### Phase 1: Infrastructure (no caller changes)

- Create `key.go`, `ttl.go`, `null.go`, `metrics.go`, `types.go`, `redis.go`
- Register `cache.NewRedisStore(rdb.Client)` in fx provider
- Existing code unchanged; new RedisStore coexists with rdb.Client

### Phase 2: Module-by-module migration

| Order | Module                            | Change                                                                 |
| ----- | --------------------------------- | ---------------------------------------------------------------------- |
| 1     | `internal/server/func.go`         | chat + online → `RedisStore.StringCache`                               |
| 2     | `internal/modules/server/cron.go` | online count → `RedisStore.IntCache`                                   |
| 3     | `pkg/crawler/crawler.go`          | sent/todo/sendtime → `RedisStore.SetCache` / `StringCache`             |
| 4     | `pkg/types/ruleset/cron/cron.go`  | filter set → `RedisStore.SetCache`                                     |
| 5     | `pkg/rdb/metrics.go`              | migrate to `RedisStore.IntCache`, deprecate old funcs                  |
| 6     | `pkg/rdb/unique.go`               | bloom → `RedisStore` internal `BloomFilter`, add TTL                   |
| 7     | `pkg/notify/rules/`               | throttle/aggregate → `RedisStore.IntCache` / `ListCache`               |
| 8     | `pkg/alarm/alarm.go`              | Ristretto cache-aside → `ObjectCache[struct{}]` or `StringCache.SetNX` |

### Phase 3: Cleanup

- Remove `pkg/rdb/metrics.go` and `pkg/rdb/unique.go` (logic migrated)
- If `rdb.Client` has no remaining external callers, unexport it
- Final state: `rdb.Client` used only internally by `RedisStore` and event bus

---

## Testing Strategy

- **Unit tests**: `*_test.go` co-located with source, TDD table-driven pattern per AGENTS.md
- **BDD specs**: Ginkgo v2 + Gomega for Redis integration (phase 2 migration)
- **Existing tests**: `pkg/cache/cache_test.go` preserved and expanded for new types
