# Provider Event Source

Date: 2026-05-23

## Summary

Bridge external provider state changes into flowbot's DataEvent system through two
mechanisms: inbound webhooks (provider pushes to flowbot) and cron polling
(flowbot periodically pulls and diffs provider data). Both paths produce
DataEvent records that flow through the existing EventEmitter chain (PostgreSQL
data_events -> event_outbox -> Redis Stream -> Pipeline engine).

## Motivation

Currently, DataEvents are only produced by `ability.Invoke()` when modules
actively perform operations. There is no passive detection of external state
changes. Users need to react to events that happen in connected services (new
GitHub star, new Miniflux entry, new Gitea issue) without an active user
interaction with flowbot.

## Design

### Architecture overview

Two independent event source channels, both converging on the existing
EventEmitter:

```
Way 1: Inbound Webhook
  Provider ──HTTP POST──> /webhook/provider/{path}
                            │
                            ▼
                     WebhookHook (signature verification)
                            │
                            ▼
                     WebhookConverter.Convert() → []DataEvent
                            │
                            ▼
                     EventEmitter (PG + Redis Stream) → Pipeline

Way 2: Cron Polling
  Cron Scheduler (per Resource interval)
       │
       ▼
  PollingResource.List(cursor) → (items, nextCursor)
       │
       ▼
  DiffKey + ContentHash compare → new/updated events
       │
       ▼
  EventEmitter (PG + Redis Stream) → Pipeline
```

All DataEvents from either path include `Source: "provider_event"` to
distinguish them from ability-invoked events.

### Directory structure

```
pkg/ability/
├── event_source.go              # Interfaces: WebhookConverter, PollingResource
├── event_source_manager.go      # EventSourceManager (Register/Start/Stop)
├── poll_scheduler.go            # Cron scheduler + cursor management
├── webhook_hook.go              # Fiber HTTP handler
├── polling_state.go             # In-memory state + PostgreSQL persistence
│
├── {ability_name}/              # Per-ability implementations
│   └── {provider_name}/
│       └── event_source.go      # WebhookConverter / PollingResource impl

internal/server/
└── router.go                    # Register /webhook/provider/* route

internal/store/
└── polling_state_store.go       # PostgreSQL polling_state table DAO
```

### Interfaces (`pkg/ability/event_source.go`)

```go
type WebhookConverter interface {
    WebhookPath() string
    GetWebhookSecret() (string, error)
    Convert(body []byte, headers map[string]string) ([]types.DataEvent, error)
}

type PollingResource interface {
    ResourceName() string
    DefaultInterval() time.Duration
    DiffKey(item any) string
    ContentHash(item any) string
    CursorField() string
    List(ctx context.Context, cursor string) (PollResult, error)
}

type PollResult struct {
    Items      []any
    NextCursor string
    HasMore    bool
}
```

Interfaces live in `pkg/ability/` (ability defines the contract).
Implementations live in `pkg/ability/{ability}/{provider}/event_source.go`.
Providers expose `GetWebhookSecret()` method on their client struct for auth.

### EventSourceManager (`pkg/ability/event_source_manager.go`)

```go
type EventSourceManager struct {
    mu         sync.RWMutex
    pollers    map[string]*pollEntry       // key: ResourceName()
    webhooks   map[string]WebhookConverter // key: WebhookPath()
    emitter    EventEmitter                // existing interface
    scheduler  *cron.Scheduler
    stateStore *PollingStateStore
    pool       *ants.PoolWithFunc          // reuses ability/pool.go pattern
    metrics    *EventSourceMetrics
}

func (m *EventSourceManager) RegisterPolling(r PollingResource, interval time.Duration)
func (m *EventSourceManager) RegisterWebhook(c WebhookConverter)
func (m *EventSourceManager) Start(ctx context.Context) error
func (m *EventSourceManager) Stop(ctx context.Context) error
func (m *EventSourceManager) WebhookHandler() fiber.Handler
```

Registration happens in `internal/modules/{name}/fx.go` via fx dependency
injection.

### Diff strategy

Each pollEntry maintains in-memory state (cursor + KnownItems):

```go
type PollingEntry struct {
    Cursor     string            // last poll cursor
    KnownItems map[string]string // DiffKey → ContentHash
    UpdatedAt  time.Time
}
```

Per-item comparison on each poll tick:

| Condition | Action |
|-----------|--------|
| DiffKey not in KnownItems | Emit `{resource}.created` event |
| DiffKey exists, different ContentHash | Emit `{resource}.updated` event |
| DiffKey exists, same ContentHash | Skip (no change) |

DataEvent IdempotencyKey is set to DiffKey for PostgreSQL-level dedup as a
secondary safeguard.

### Webhook Hook (`pkg/ability/webhook_hook.go`)

Route: `POST /webhook/provider/{path}`

```go
func (m *EventSourceManager) WebhookHandler() fiber.Handler {
    return func(c fiber.Ctx) error {
        path := c.Params("*")
        converter := m.webhooks[path]
        if converter == nil { return 404 }
        secret, _ := converter.GetWebhookSecret()
        if err := verifySignature(c, body, secret); err != nil { return 401 }
        events, err := converter.Convert(body, headers)
        if err != nil { return 400 }
        for _, ev := range events {
            m.pool.Submit(func() { m.emitter.Emit(ev) })
        }
        return c.SendStatus(202)
    }
}
```

Signature verification: HMAC-SHA256 (`X-Hub-Signature-256`) or plain token
(`X-Webhook-Token`). No verification when secret is empty (development only).

### Secret configuration

```yaml
# flowbot.yaml
providers:
  github:
    client_id: "xxx"
    client_secret: "xxx"
    webhook_secret: "my-hmac-key"
```

```go
// pkg/providers/github/github.go
func (g *Github) GetWebhookSecret() (string, error) {
    return providers.GetConfig("github", "webhook_secret")
}
```

### Polling state persistence (`pkg/ability/polling_state.go`)

Memory-first with periodic PostgreSQL flush:

```sql
CREATE TABLE polling_state (
    resource_name TEXT PRIMARY KEY,
    cursor        TEXT NOT NULL DEFAULT '',
    known_hashes  JSONB NOT NULL DEFAULT '{}',
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Persistence timing:
- After each poll tick: update in-memory, mark dirty
- Background goroutine: flush dirty entries every 5 minutes
- Stop(): force flush all dirty entries
- Start(): Load from PG, truncate known_hashes to last 1000 entries

### Error handling

Per-poll errors:
- `context.DeadlineExceeded`: skip, retry next tick without updating cursor
- Provider API errors: increment consecutive failure counter, log warning at 3+
  consecutive failures, log error at failure. Exponential back-off via skip
  counter.
- Success resets consecutive failure counter to 0

Webhook errors:
- Signature mismatch: return 401, do not emit
- Converter error (malformed payload): return 400
- Emit failure: log + metrics, HTTP already returned 202

Poll timeout: each `List()` call receives a context with deadline set to
`interval / 2` (minimum 30 seconds).

### Lifecycle (fx)

```go
fx.Invoke(func(lc fx.Lifecycle, mgr *ability.EventSourceManager) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            mgr.Start(ctx)  // Load state from PG, register cron jobs
            return nil
        },
        OnStop: func(ctx context.Context) error {
            mgr.Stop(ctx)   // Flush all state to PG, stop cron scheduler
            return nil
        },
    })
})
```

### Metrics (Prometheus)

| Metric | Labels | Description |
|--------|--------|-------------|
| `event_source_poll_total` | resource, status | Poll completions |
| `event_source_poll_events` | resource, event_type | Events emitted per poll |
| `event_source_poll_duration` | resource | Poll execution time |
| `event_source_poll_errors` | resource | Failed polls |
| `event_source_webhook_total` | path, status | Webhook requests |
| `event_source_webhook_events` | path | Events emitted per webhook |
| `event_source_state_flush_duration` | - | PG flush duration |

### Testing strategy

| Layer | Type | Coverage |
|-------|------|----------|
| `pkg/ability/event_source_test.go` | Unit (TDD) | Interface types, PollResult |
| `pkg/ability/event_source_manager_test.go` | Unit (TDD) | Register/Start/Stop, duplicate registration, concurrent registration |
| `pkg/ability/poll_scheduler_test.go` | Unit (TDD) | Cron tick, cursor update, diff dedup, content change detection, consecutive failure backoff |
| `pkg/ability/webhook_hook_test.go` | Unit (TDD) | Valid/invalid signatures, 404, empty events, converter errors, malformed payload |
| `pkg/ability/polling_state_test.go` | Unit (TDD) | Load/Update/Flush, recovery, knownItems truncation |
| `specs/provider_event_source/` | BDD (Ginkgo) | Full webhook flow, full polling flow, state persistence + recovery, error isolation |

All tests use the table-driven pattern (`for _, tt := range tests { t.Run(...) }`).
BDD uses Ginkgo v2 with `SynchronizedBeforeSuite` for database isolation.

Mocking: `EventEmitter` interface mock, `PollingResource` stub, `WebhookConverter`
stub, `Clock` interface (reuse `pkg/pipeline/clock.go` pattern).

### Anti-pattern compliance

- Providers never emit DataEvents, call Hub, or call Pipeline
- Modules never import `pkg/providers/*` directly; use ability interfaces
- All `DataEvent` records persist to PostgreSQL `data_events` table; Redis
  Stream is not the sole event store
- Idempotency via `IdempotencyKey` on all emitted events

## Scope

This design covers the framework and abstraction. Per-provider implementations
(e.g., `pkg/ability/bookmark/github/event_source.go`) are out of scope and will
be handled in separate design docs.

## References

- [Pipeline Webhook Trigger](2026-05-22-pipeline-webhook-trigger-design.md)
- [Pipeline Cron Trigger](2026-05-22-pipeline-cron-trigger-design.md)
- [Ability Event Worker Pool](2026-05-21-ability-event-worker-pool-design.md)
- [Bulkhead Isolation](2026-05-21-bulkhead-isolation-design.md)
- Architecture: `docs/architecture/README.md`
