# Unify Redis Client & Pool Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge three independent Redis clients into one, and add full connection pool configuration to the config struct.

**Architecture:** `rdb.NewClient` becomes the sole `*redis.Client` factory reading pool config from `config.Redis`. `NewSubscriber` and `NewPublisher` accept the shared client via fx injection instead of creating their own. `pkg/event/redis.go` is deleted. `NewRouter` drops its unused `*redis.Client` parameter.

**Tech Stack:** go-redis v9, Uber fx, Watermill redisstream

---

## File Structure

| File                         | Responsibility                                                                                                   |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `pkg/config/config.go`       | `Redis` struct — add pool fields                                                                                 |
| `pkg/rdb/rdb.go`             | `NewClient` — apply pool config when constructing client. `Shutdown` unchanged.                                  |
| `pkg/event/redis.go`         | **Delete** — `newRedisClient()` removed                                                                          |
| `pkg/event/pubsub.go`        | `NewSubscriber`, `NewPublisher` — accept `*redis.Client` param. `NewRouter` — drop unused `*redis.Client` param. |
| `internal/server/fx.go`      | No changes — fx auto-resolves by type                                                                            |
| `docs/reference/config.yaml` | Add pool field examples                                                                                          |
| `pkg/config/config_test.go`  | Add pool config test cases                                                                                       |
| `pkg/rdb/rdb_test.go`        | New file — test pool options from config                                                                         |
| `pkg/event/pubsub_test.go`   | Add constructor tests with injected client                                                                       |

---

### Task 1: Add pool fields to config.Redis struct

**Files:**

- Modify: `pkg/config/config.go:253-262`
- Modify: `pkg/config/config_test.go:142-188`

- [ ] **Step 1: Add `"time"` to imports in config.go**

Open `pkg/config/config.go`. After `"strings"` on line 10, add `"time"`.

```go
import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	goYaml "github.com/goccy/go-yaml"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
```

- [ ] **Step 2: Replace the `Redis` struct with the expanded version**

Replace lines 253-262 of `pkg/config/config.go`:

```go
// Redis stores connection and pool configuration for the Redis client.
type Redis struct {
	// Redis host
	Host string `json:"host" yaml:"host" mapstructure:"host"`
	// Redis port
	Port int `json:"port" yaml:"port" mapstructure:"port"`
	// Redis database
	DB int `json:"db" yaml:"db" mapstructure:"db"`
	// Redis password
	Password string `json:"password" yaml:"pass" mapstructure:"password"`
	// Maximum number of connections in the pool (0 = go-redis default: 10*GOMAXPROCS)
	PoolSize int `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"`
	// Minimum number of idle connections maintained in the pool (0 = default: none)
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	// Maximum number of retries before giving up (0 = default: 3)
	MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`
	// Minimum backoff between retries (0 = default: 8ms)
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff" mapstructure:"min_retry_backoff"`
	// Maximum backoff between retries (0 = default: 512ms)
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff" mapstructure:"max_retry_backoff"`
	// Dial timeout for establishing new connections (0 = default: 5s)
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout" mapstructure:"dial_timeout"`
	// Timeout for socket reads (0 = fallback to 60s for backward compatibility)
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`
	// Timeout for socket writes (0 = fallback to 60s for backward compatibility)
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"`
	// Timeout for waiting for a connection from the pool (0 = default: ReadTimeout + 1s)
	PoolTimeout time.Duration `json:"pool_timeout" yaml:"pool_timeout" mapstructure:"pool_timeout"`
	// Maximum idle time for a connection before closing (0 = default: 30min)
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
	// Maximum lifetime of a connection (0 = default: no limit)
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	// Use FIFO (first-in-first-out) instead of LIFO for pool connections
	PoolFIFO bool `json:"pool_fifo" yaml:"pool_fifo" mapstructure:"pool_fifo"`
}
```

- [ ] **Step 3: Write config test for pool fields**

Replace the `TestRedis` function in `pkg/config/config_test.go` (lines 142-188) with expanded tests:

```go
func TestRedis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		redis    Redis
		wantHost string
		wantPort int
		wantDB   int
		wantPass string
	}{
		{
			name:     "full config",
			redis:    Redis{Host: "localhost", Port: 6379, DB: 0, Password: "secret"},
			wantHost: "localhost",
			wantPort: 6379,
			wantDB:   0,
			wantPass: "secret",
		},
		{
			name:     "remote redis with different db",
			redis:    Redis{Host: "redis.example.com", Port: 6380, DB: 1, Password: ""},
			wantHost: "redis.example.com",
			wantPort: 6380,
			wantDB:   1,
			wantPass: "",
		},
		{
			name:     "zero value config",
			redis:    Redis{},
			wantHost: "",
			wantPort: 0,
			wantDB:   0,
			wantPass: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantHost, tt.redis.Host)
			assert.Equal(t, tt.wantPort, tt.redis.Port)
			assert.Equal(t, tt.wantDB, tt.redis.DB)
			assert.Equal(t, tt.wantPass, tt.redis.Password)
		})
	}
}

func TestRedisPoolConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		redis         Redis
		wantPoolSize  int
		wantMinIdle   int
		wantMaxRetry  int
		wantFIFO      bool
		wantReadTO    time.Duration
		wantWriteTO   time.Duration
	}{
		{
			name: "all pool fields set",
			redis: Redis{
				PoolSize:        20,
				MinIdleConns:    5,
				MaxRetries:      5,
				PoolFIFO:        true,
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				MinRetryBackoff: 100 * time.Millisecond,
				MaxRetryBackoff: 5 * time.Second,
				DialTimeout:     3 * time.Second,
				PoolTimeout:     10 * time.Second,
				ConnMaxIdleTime: 10 * time.Minute,
				ConnMaxLifetime: 1 * time.Hour,
			},
			wantPoolSize: 20,
			wantMinIdle:  5,
			wantMaxRetry: 5,
			wantFIFO:     true,
			wantReadTO:   30 * time.Second,
			wantWriteTO:  30 * time.Second,
		},
		{
			name: "partial pool config",
			redis: Redis{
				PoolSize:     10,
				MinIdleConns: 2,
			},
			wantPoolSize: 10,
			wantMinIdle:  2,
			wantMaxRetry: 0,
			wantFIFO:     false,
			wantReadTO:   0,
			wantWriteTO:  0,
		},
		{
			name: "zero pool config uses defaults",
			redis: Redis{
				Host: "localhost",
				Port: 6379,
			},
			wantPoolSize: 0,
			wantMinIdle:  0,
			wantMaxRetry: 0,
			wantFIFO:     false,
			wantReadTO:   0,
			wantWriteTO:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantPoolSize, tt.redis.PoolSize)
			assert.Equal(t, tt.wantMinIdle, tt.redis.MinIdleConns)
			assert.Equal(t, tt.wantMaxRetry, tt.redis.MaxRetries)
			assert.Equal(t, tt.wantFIFO, tt.redis.PoolFIFO)
			assert.Equal(t, tt.wantReadTO, tt.redis.ReadTimeout)
			assert.Equal(t, tt.wantWriteTO, tt.redis.WriteTimeout)
		})
	}
}
```

- [ ] **Step 4: Run config tests**

```bash
go test ./pkg/config/ -run "TestRedis" -v
```

Expected: PASS (4 subtests: 3 existing + 3 new pool test subtests)

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat: add Redis connection pool fields to config struct"
```

---

### Task 2: Build single client factory with pool config in rdb.NewClient

**Files:**

- Modify: `pkg/rdb/rdb.go`
- Create: `pkg/rdb/rdb_test.go`

- [ ] **Step 1: Add a `redisOptions` helper function and update `NewClient`**

Replace the entire content of `pkg/rdb/rdb.go`:

```go
// Package rdb provides a singleton Redis client with connection pool configuration.
package rdb

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// Client is the singleton Redis client, provided by NewClient via fx.
var Client *redis.Client

// NewClient creates and returns a single Redis client configured from config.App.Redis.
// Connection pool parameters use go-redis defaults when set to zero, except ReadTimeout
// and WriteTimeout which fall back to 60s for backward compatibility.
func NewClient(lc fx.Lifecycle, _ *config.Type) (*redis.Client, error) {
	addr := net.JoinHostPort(config.App.Redis.Host, strconv.Itoa(config.App.Redis.Port))
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		return nil, fmt.Errorf("redis config error")
	}
	Client = redis.NewClient(redisOptions(config.App.Redis))
	if err := redisotel.InstrumentTracing(Client); err != nil {
		return nil, fmt.Errorf("failed to instrument redis with tracing: %w", err)
	}
	s := Client.Ping(context.Background())
	_, err := s.Result()
	if err != nil {
		return nil, fmt.Errorf("redis server error %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			Shutdown(ctx)
			return nil
		},
	})

	return Client, nil
}

// redisOptions builds a go-redis Options from the config, applying fallback defaults
// for ReadTimeout and WriteTimeout when they are zero.
func redisOptions(cfg config.Redis) *redis.Options {
	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 60 * time.Second
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 60 * time.Second
	}

	return &redis.Options{
		Addr:            net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		PoolFIFO:        cfg.PoolFIFO,
	}
}

// Shutdown gracefully closes the Redis client with a 5-second timeout.
func Shutdown(ctx context.Context) {
	if Client == nil {
		flog.Warn("redis not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := Client.Ping(ctx).Result()
	if err == nil {
		err = Client.Close()
		if err != nil {
			flog.Error(fmt.Errorf("failed to close redis connection: %w", err))
			return
		}
		flog.Info("redis stopped")
	} else {
		flog.Warn("redis connection already lost: %v", err)
	}
}
```

- [ ] **Step 2: Write unit tests for `redisOptions`**

Create `pkg/rdb/rdb_test.go`:

```go
package rdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/config"
)

func TestRedisOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         config.Redis
		wantAddr    string
		wantDB      int
		wantPool    int
		wantMinIdle int
		wantReadTO  time.Duration
		wantWriteTO time.Duration
		wantRetries int
		wantFIFO    bool
	}{
		{
			name: "all pool fields set explicitly",
			cfg: config.Redis{
				Host:            "redis.example.com",
				Port:            6379,
				DB:              2,
				Password:        "secret",
				PoolSize:        20,
				MinIdleConns:    5,
				MaxRetries:      5,
				MinRetryBackoff: 100 * time.Millisecond,
				MaxRetryBackoff: 5 * time.Second,
				DialTimeout:     3 * time.Second,
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				PoolTimeout:     10 * time.Second,
				ConnMaxIdleTime: 10 * time.Minute,
				ConnMaxLifetime: 1 * time.Hour,
				PoolFIFO:        true,
			},
			wantAddr:    "redis.example.com:6379",
			wantDB:      2,
			wantPool:    20,
			wantMinIdle: 5,
			wantReadTO:  30 * time.Second,
			wantWriteTO: 30 * time.Second,
			wantRetries: 5,
			wantFIFO:    true,
		},
		{
			name: "zero pool config uses go-redis defaults",
			cfg: config.Redis{
				Host:     "localhost",
				Port:     6379,
				DB:       0,
				Password: "",
			},
			wantAddr:    "localhost:6379",
			wantDB:      0,
			wantPool:    0,
			wantMinIdle: 0,
			wantReadTO:  60 * time.Second,
			wantWriteTO: 60 * time.Second,
			wantRetries: 0,
			wantFIFO:    false,
		},
		{
			name: "zero read and write timeout falls back to 60s",
			cfg: config.Redis{
				Host:         "127.0.0.1",
				Port:         6380,
				DB:           1,
				Password:     "pwd",
				ReadTimeout:  0,
				WriteTimeout: 0,
			},
			wantAddr:    "127.0.0.1:6380",
			wantDB:      1,
			wantPool:    0,
			wantMinIdle: 0,
			wantReadTO:  60 * time.Second,
			wantWriteTO: 60 * time.Second,
			wantRetries: 0,
			wantFIFO:    false,
		},
		{
			name: "partial pool config with subset of fields",
			cfg: config.Redis{
				Host:            "10.0.0.1",
				Port:            6379,
				DB:              3,
				Password:        "pass",
				PoolSize:        50,
				MinIdleConns:    10,
				ConnMaxLifetime: 30 * time.Minute,
			},
			wantAddr:    "10.0.0.1:6379",
			wantDB:      3,
			wantPool:    50,
			wantMinIdle: 10,
			wantReadTO:  60 * time.Second,
			wantWriteTO: 60 * time.Second,
			wantRetries: 0,
			wantFIFO:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := redisOptions(tt.cfg)

			assert.Equal(t, tt.wantAddr, opts.Addr)
			assert.Equal(t, tt.wantDB, opts.DB)
			assert.Equal(t, tt.wantPool, opts.PoolSize)
			assert.Equal(t, tt.wantMinIdle, opts.MinIdleConns)
			assert.Equal(t, tt.wantReadTO, opts.ReadTimeout)
			assert.Equal(t, tt.wantWriteTO, opts.WriteTimeout)
			assert.Equal(t, tt.wantRetries, opts.MaxRetries)
			assert.Equal(t, tt.wantFIFO, opts.PoolFIFO)
			assert.Equal(t, tt.cfg.Password, opts.Password)
		})
	}
}
```

- [ ] **Step 3: Run rdb tests**

```bash
go test ./pkg/rdb/ -v -run TestRedisOptions
```

Expected: PASS (4 subtests)

- [ ] **Step 4: Commit**

```bash
git add pkg/rdb/rdb.go pkg/rdb/rdb_test.go
git commit -m "feat: apply Redis pool config in NewClient, add redisOptions helper"
```

---

### Task 3: Delete event/redis.go, inject client into Subscriber/Publisher, clean up NewRouter

**Files:**

- Delete: `pkg/event/redis.go`
- Modify: `pkg/event/pubsub.go`
- Modify: `pkg/event/pubsub_test.go`

- [ ] **Step 1: Delete `pkg/event/redis.go`**

```bash
rm pkg/event/redis.go
```

- [ ] **Step 2: Update `pkg/event/pubsub.go` — inject `*redis.Client` into Subscriber and Publisher, drop unused param from NewRouter**

Replace the entire content of `pkg/event/pubsub.go`:

```go
// Package event provides Watermill-based publish/subscribe infrastructure backed by Redis Streams.
package event

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/trace"
)

var logger = flog.WatermillLogger

// NewSubscriber creates a Watermill Redis Stream subscriber using the shared Redis client.
func NewSubscriber(lc fx.Lifecycle, client *redis.Client) (message.Subscriber, error) {
	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:       client,
			Unmarshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis subscriber: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
		OnStop: func(_ context.Context) error {
			return subscriber.Close()
		},
	})

	return subscriber, err
}

// Publisher is the global Watermill publisher, provided by NewPublisher via fx.
var Publisher message.Publisher

// NewPublisher creates a Watermill Redis Stream publisher using the shared Redis client.
func NewPublisher(lc fx.Lifecycle, client *redis.Client) (message.Publisher, error) {
	var err error
	Publisher, err = redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     client,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
		OnStop: func(_ context.Context) error {
			return Publisher.Close()
		},
	})

	return Publisher, err
}

// NewRouter creates a Watermill message router with standard middleware.
func NewRouter(_ *sdktrace.TracerProvider) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Timeout(10*time.Minute),
		Retry{
			MaxRetries:          3,
			InitialInterval:     1 * time.Second,
			MaxInterval:         30 * time.Second,
			Multiplier:          2.0,
			MaxElapsedTime:      2 * time.Minute,
			RandomizationFactor: 0.5,
			OnRetryHook: func(retryNum int, delay time.Duration) {
				flog.Info("Retry attempt #%d, waiting %v before next retry", retryNum, delay)
			},
			Logger: logger,
		}.Middleware,
		middleware.Recoverer,
	)

	router.AddMiddleware(TraceConsumerMiddleware())

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(message *message.Message) ([]*message.Message, error) {
			flog.Debug("executing handler specific middleware for %s", message.UUID)
			stats.EventTotalCounter().Inc()
			return h(message)
		}
	})

	return router, nil
}

// NewMessage creates a Watermill message from the given payload, marshaled as JSON.
func NewMessage(payload any) (*message.Message, error) {
	data, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	middleware.SetCorrelationID(watermill.NewShortUUID(), msg)

	return msg, nil
}

// PublishMessage publishes a message to the given topic with OpenTelemetry tracing.
func PublishMessage(ctx context.Context, topic string, payload any) error {
	msg, err := NewMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to new message: %w", err)
	}

	_, publishSpan := trace.StartSpan(ctx, "event.publish "+topic,
		attribute.String("messaging.destination", topic),
		attribute.String("messaging.message.id", msg.UUID),
	)
	defer publishSpan.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		msg.Metadata.Set(k, v)
	}
	msg.Metadata.Set("x-otel-topic", topic)

	err = Publisher.Publish(topic, msg)
	if err != nil {
		publishSpan.RecordError(err)
		publishSpan.SetStatus(codes.Error, err.Error())
	}
	return err
}

// TraceConsumerMiddleware returns a Watermill middleware that extracts OTel trace context
// from message metadata and creates a consumer span for each incoming message.
func TraceConsumerMiddleware() message.HandlerMiddleware {
	prop := otel.GetTextMapPropagator()
	tracer := otel.Tracer("watermill")

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			carrier := propagation.MapCarrier{}
			for k, v := range msg.Metadata {
				carrier.Set(k, v)
			}
			ctx := prop.Extract(msg.Context(), carrier)

			topic := ""
			if t := msg.Metadata.Get("x-otel-topic"); t != "" {
				topic = t
				delete(msg.Metadata, "x-otel-topic")
			}

			spanName := "event.receive"
			if topic != "" {
				spanName = "event.receive " + topic
			}

			ctx, span := tracer.Start(ctx, spanName)
			span.SetAttributes(
				attribute.String("messaging.operation", "receive"),
				attribute.String("messaging.message.id", msg.UUID),
			)
			if topic != "" {
				span.SetAttributes(attribute.String("messaging.destination", topic))
			}
			msg.SetContext(ctx)
			defer span.End()

			return h(msg)
		}
	}
}
```

- [ ] **Step 3: Add `NewRouter` signature verification test**

Add the following test function to `pkg/event/pubsub_test.go` (at the end of the file). `NewSubscriber` and `NewPublisher` signature changes are verified by the `go build ./...` step below; they require a live Redis connection to run.

```go
func TestNewRouterSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "NewRouter no longer accepts *redis.Client"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// nil TracerProvider is safe — the parameter is unused (blank identifier).
			router, err := NewRouter(nil)
			assert.NoError(t, err)
			assert.NotNil(t, router)
			_ = router.Close()
		})
	}
}
```

- [ ] **Step 4: Run event tests**

```bash
go test ./pkg/event/ -v
```

Expected: PASS (all existing tests + new signature test; subscriber/publisher tests skipped)

- [ ] **Step 5: Ensure the project compiles**

```bash
go build ./...
```

Expected: no compilation errors

- [ ] **Step 6: Commit**

```bash
git add pkg/event/redis.go pkg/event/pubsub.go pkg/event/pubsub_test.go
git commit -m "refactor: merge Redis clients into single instance, inject into pubsub"
```

---

### Task 4: Update reference config and final verification

**Files:**

- Modify: `docs/reference/config.yaml`

- [ ] **Step 1: Add pool config example to reference config**

Replace lines 53-58 of `docs/reference/config.yaml`:

```yaml
# Redis configuration
redis:
  host: 127.0.0.1
  port: 6379
  db: 0
  password:
  # Connection pool configuration (all optional — zero means go-redis default)
  # pool_size: 20            # max connections (default: 10*GOMAXPROCS)
  # min_idle_conns: 5        # min idle connections (default: 0)
  # max_retries: 3           # max retries on failure (default: 3)
  # min_retry_backoff: 8ms   # min backoff between retries (default: 8ms)
  # max_retry_backoff: 512ms # max backoff between retries (default: 512ms)
  # dial_timeout: 5s         # timeout for new connections (default: 5s)
  # read_timeout: 60s        # socket read timeout (default: 60s)
  # write_timeout: 60s       # socket write timeout (default: 60s)
  # pool_timeout: 61s        # wait timeout for pool connection (default: read_timeout + 1s)
  # conn_max_idle_time: 30m  # max idle time before closing (default: 30min)
  # conn_max_lifetime: 0     # max connection lifetime, 0 = no limit
  # pool_fifo: false         # FIFO instead of LIFO pool ordering
```

- [ ] **Step 2: Run full test suite**

```bash
go test ./pkg/config/ ./pkg/rdb/ ./pkg/event/ -v
```

Expected: all tests PASS

- [ ] **Step 3: Run lint**

```bash
go tool task lint
```

Expected: no lint errors

- [ ] **Step 4: Run build**

```bash
go tool task build
```

Expected: build succeeds

- [ ] **Step 5: Commit**

```bash
git add docs/reference/config.yaml
git commit -m "docs: add Redis connection pool configuration example"
```
