# Unify Redis Client & Pool Configuration

## Context

The codebase creates three independent `*redis.Client` instances, all connecting to the same Redis server with identical connection parameters but no pool tuning:

1. **`pkg/rdb/rdb.go:NewClient()`** — global `rdb.Client` used for caching (`cache.RedisStore`), distributed locking (`pkg/locker`), and direct key scanning (`internal/modules/server/command.go`, `internal/modules/server/cron.go`).
2. **`pkg/event/pubsub.go:NewSubscriber()`** — calls unexported `newRedisClient()` in `pkg/event/redis.go`, creates a dedicated client for Watermill Redis Stream subscriber.
3. **`pkg/event/pubsub.go:NewPublisher()`** — calls the same `newRedisClient()`, creates a third client for Watermill Redis Stream publisher.

## Motivation

- **Wasted connections**: Three separate pools competing for the same Redis server.
- **No pool tuning**: Both use go-redis v9 defaults. No way to configure `PoolSize`, `MinIdleConns`, retry behavior, or idle connection management.
- **Code duplication**: `rdb.go` and `event/redis.go` contain nearly identical client creation logic.
- **`NewRouter` dead parameter**: `NewRouter(_ *redis.Client, ...)` accepts a `*redis.Client` it never uses.

## Design

### 1. Config: pool fields on `config.Redis`

Add optional connection pool fields to `pkg/config/config.go`. All fields are zero-value defaults — a zero means "use go-redis default." Exceptions: `ReadTimeout` and `WriteTimeout` fall back to the current hardcoded value (60s) when zero, preserving backward compatibility.

```go
type Redis struct {
    Host     string
    Port     int
    DB       int
    Password string
    // Connection pool
    PoolSize        int
    MinIdleConns    int
    MaxRetries      int
    MinRetryBackoff time.Duration
    MaxRetryBackoff time.Duration
    DialTimeout     time.Duration
    ReadTimeout     time.Duration  // fallback: 60s
    WriteTimeout    time.Duration  // fallback: 60s
    PoolTimeout     time.Duration
    ConnMaxIdleTime time.Duration
    ConnMaxLifetime time.Duration
    PoolFIFO        bool
}
```

### 2. Single client factory: `pkg/rdb/rdb.go`

`NewClient` reads all pool config and builds one `*redis.Client`. It is the sole client factory.

### 3. Delete `pkg/event/redis.go`

The `newRedisClient()` function is removed. Client creation lives exclusively in `rdb.go`.

### 4. Inject client into Subscriber/Publisher

Change signatures to accept `*redis.Client` via fx injection:

```go
func NewSubscriber(lc fx.Lifecycle, client *redis.Client) (message.Subscriber, error)
func NewPublisher(lc fx.Lifecycle, client *redis.Client) (message.Publisher, error)
```

Both receive the same instance that `rdb.NewClient` provides. Watermill `Close()` does not close the underlying client — only clean up Watermill internals. The single client is shut down once by `rdb.Shutdown`.

### 5. Clean up `NewRouter`

Remove the unused first parameter:

```go
func NewRouter(_ *sdktrace.TracerProvider) (*message.Router, error)
```

### Shutdown order

fx stops hooks in reverse order. The Watermill subscriber and publisher close their internal goroutines first, then `rdb.Shutdown` closes the single Redis client. No race: by the time the client closes, no Watermill goroutine is still making calls.

## Files changed

| File                         | Change                                                              |
| ---------------------------- | ------------------------------------------------------------------- |
| `pkg/config/config.go`       | Add pool fields to `Redis` struct                                   |
| `pkg/rdb/rdb.go`             | Apply pool config in `NewClient`                                    |
| `pkg/event/redis.go`         | Delete                                                              |
| `pkg/event/pubsub.go`        | Accept `*redis.Client` via injection; drop unused `NewRouter` param |
| `docs/reference/config.yaml` | Add pool field examples with defaults                               |
| `pkg/config/config_test.go`  | Add pool config test cases                                          |
| `pkg/rdb/rdb_test.go`        | Add `NewClient` pool config tests                                   |
| `pkg/event/pubsub_test.go`   | Add constructor tests with injected client                          |

## Risk assessment

- **Single pool contention**: Cache, lock, and pub/sub share one pool. With a properly sized `PoolSize` (default 10\*GOMAXPROCS) and `MinIdleConns > 0`, this is not a problem at expected throughput.
- **Backward compatibility**: Zero pool config → go-redis defaults or current hardcoded 60s timeouts. No behavior change for existing deployments.
- **Shutdown ordering**: Verified — Watermill subscriber/publisher close before the Redis client.

## Testing

- **Unit**: `pkg/rdb/rdb_test.go` — verify pool options are applied correctly from config, verify zero-value defaults.
- **Unit**: `pkg/config/config_test.go` — round-trip config struct tests including new pool fields.
- **Unit**: `pkg/event/pubsub_test.go` — verify `NewSubscriber`/`NewPublisher` constructors with injected mock client.
- **BDD**: Existing specs cover the full app startup via fx; no new BDD tests needed since this is an internal wiring change.
