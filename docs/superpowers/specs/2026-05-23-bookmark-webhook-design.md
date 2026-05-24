# Bookmark Webhook Support

## Motivation

The bookmark ability currently supports REST CRUD operations through the Karakeep adapter, but has no inbound webhook support. Adding a WebhookConverter allows external services (primarily Karakeep) to push bookmark events into flowbot's event pipeline, enabling real-time reactivity (notifications, pipeline triggers, workflow automation) when bookmarks are created, updated, archived, or deleted.

## Design Overview

Add a Karakeep-specific `WebhookConverter` implementation in `pkg/ability/bookmark/karakeep/`, registered with the `EventSourceManager` during the bookmark module's `Bootstrap()`. The converter accepts webhook POSTs at `/webhook/provider/karakeep/events`, authenticates via `Authorization: Bearer <token>`, and transforms Karakeep webhook payloads into `types.DataEvent` records.

Registration uses a global `GetEventSourceManager()` accessor (following the `GetEventPool()` pattern). To avoid a nil-dereference race, `initPipeline` must run **before** `handleModules` in `internal/server/fx.go` so that the manager is created and stored in the global before any module's `Bootstrap()` runs.

## Architecture

```
POST /webhook/provider/karakeep/events
  Authorization: Bearer <webhook_token>
  Body: {"event_type": "...", "data": {...}, "timestamp": "..."}
    -> EventSourceManager.WebhookHandler()
      -> Webhook.VerifySignature(headers, body)     // Bearer token validation
      -> Webhook.Convert(body)                       // payload -> DataEvent[]
      -> poolSubmit(emitter(DataEvent))              // async emit to pipeline
```

The webhook converter lives alongside the existing Karakeep adapter (`pkg/ability/bookmark/karakeep/`), keeping all provider-specific logic co-located. The centralized `EventSourceManager` handles dispatch, signature verification, error responses, and async event emission — the converter only needs to implement 3 methods.

## Files & Changes

### New files

| File | Purpose |
|------|---------|
| `pkg/ability/bookmark/karakeep/webhook.go` | `Webhook` struct implementing `ability.WebhookConverter` |
| `pkg/ability/bookmark/karakeep/webhook_test.go` | Table-driven unit tests |

### Modified files

| File | Change |
|------|--------|
| `pkg/types/event.go` | Add `EventBookmarkUpdated`, `EventBookmarkDeleted` event constants |
| `pkg/ability/event_source_manager.go` | Add `SetEventSourceManager` / `GetEventSourceManager` global accessor (follows `pool.go` pattern) |
| `internal/server/fx.go` | Move `initPipeline` before `handleModules` in `fx.Invoke` so the manager is set before module `Bootstrap()` |
| `internal/server/pipeline.go` | Call `ability.SetEventSourceManager(srcMgr)` after creating the manager |
| `pkg/providers/karakeep/types.go` | Add `WebhookPayload` struct |
| `pkg/providers/karakeep/karakeep.go` | Add `WebhookTokenKey` constant and `GetWebhookToken()` config reader |
| `internal/modules/bookmark/module.go` | Register WebhookConverter with EventSourceManager in `Bootstrap()` |
| `docs/reference/config.yaml` | Add `webhook_token` to `vendors.karakeep` |
| `flowbot.yaml` | Add `webhook_token` to karakeep config |

## Data Model

### WebhookPayload (pkg/providers/karakeep/types.go)

```go
type WebhookPayload struct {
    EventType string   `json:"event_type"`
    Timestamp string   `json:"timestamp"`
    Data      Bookmark `json:"data"`
}
```

Reuses the existing `Bookmark` type. The `event_type` field carries values matching `EventBookmark*` constants.

**Note:** The `Data` field typed as `Bookmark` assumes Karakeep webhooks embed the full bookmark object. If the actual Karakeep webhook payload sends a subset (e.g., just an ID) or a different shape, define a separate `WebhookBookmarkData` type with only the fields Karakeep actually sends. Verify against the Karakeep webhook documentation before finalizing.

### Event constants (pkg/types/event.go)

Added to existing `EventBookmarkCreated` and `EventBookmarkArchived`:

```go
EventBookmarkUpdated = "bookmark.updated"
EventBookmarkDeleted = "bookmark.deleted"
```

### DataEvent output

| DataEvent field | Source |
|-----------------|--------|
| `EventID` | `types.Id()` |
| `EventType` | `payload.EventType` |
| `Source` | `"karakeep_webhook"` |
| `Capability` | `"bookmark"` |
| `Operation` | Inferred from event type: `"created"`, `"updated"`, `"archived"`, or `"deleted"` |
| `EntityID` | `payload.Data.Id` |
| `IdempotencyKey` | `payload.Data.Id` |
| `Backend` | `"karakeep"` |
| `Data` | `types.KV{"bookmark": toBookmark(payload.Data), "event_type": payload.EventType}` |

## WebhookConverter Implementation

### Webhook struct

Uses lazy token evaluation via a `getToken` closure, following the `pkg/ability/example/example/webhook.go` pattern. The token is read from provider config at verification time, not at construction time.

```go
type Webhook struct {
    getToken func() string
}

func NewWebhook() *Webhook {
    return &Webhook{
        getToken: karakeep.GetWebhookToken,
    }
}

// Compile-time interface check
var _ ability.WebhookConverter = (*Webhook)(nil)
```

### WebhookPath

Returns `"karakeep/events"` — the URL path segment under `/webhook/provider/`.

### VerifySignature

Bearer token validation against the configured `webhook_token`:

1. Read the configured token via `w.getToken()`
2. If `webhook_token` config is empty → `types.Errorf(types.ErrUnauthorized, "webhook token not configured")` (fail closed, matching the example pattern)
3. Extract `Authorization` header
4. Validate `Bearer ` prefix
5. Compare token value to configured `webhook_token`
6. Mismatch → `types.Errorf(types.ErrUnauthorized, "...")`

### Convert

1. Parse body as `provider.WebhookPayload` using `sonic.Unmarshal`
2. Invalid JSON → `types.Errorf(types.ErrInvalidArgument, "...")`
3. Derive `Operation` from `payload.EventType` (strip the `"bookmark."` prefix to get `"created"`, `"updated"`, `"archived"`, or `"deleted"`)
4. Build single `DataEvent` populating all fields per the DataEvent output table above
5. Return `[]types.DataEvent{ev}`

## Global EventSourceManager Accessor

Since the `EventSourceManager` is created in `internal/server/pipeline.go` as a local variable, modules need a way to register webhooks. A global accessor follows the existing `GetEventPool()` / `InitEventPool()` pattern in `pkg/ability/pool.go`:

```go
// pkg/ability/event_source_manager.go

var (
    globalSrcMgr   *EventSourceManager
    globalSrcMgrMu sync.Mutex
)

func SetEventSourceManager(m *EventSourceManager) {
    globalSrcMgrMu.Lock()
    defer globalSrcMgrMu.Unlock()
    globalSrcMgr = m
}

func GetEventSourceManager() *EventSourceManager {
    globalSrcMgrMu.Lock()
    defer globalSrcMgrMu.Unlock()
    return globalSrcMgr
}
```

`internal/server/pipeline.go` calls `ability.SetEventSourceManager(srcMgr)` after `NewEventSourceManager(...)`.

## Registration Flow

In `internal/modules/bookmark/module.go` `Bootstrap()`:

1. Check `Enabled` flag (existing logic from `Init()`)
2. Create `karakeep.NewWebhook()` (token is read lazily at verification time — no config call needed at registration)
3. Get the manager via `ability.GetEventSourceManager()`
4. If nil → return error (manager must be set by `initPipeline` before `Bootstrap()`)
5. Call `mgr.RegisterWebhook(webhook)`

## Testing

### Unit tests (webhook_test.go)

Table-driven tests following the `pkg/ability/example/webhook_test.go` pattern:

- **TestWebhookPath**: 3+ cases verifying path is always `"karakeep/events"`
- **TestVerifySignature**: 5+ cases:
  - valid Bearer token → no error
  - missing Authorization header → error
  - wrong token → error
  - missing Bearer prefix → error
  - empty configured token → error (fail closed)
- **TestConvert**: 5+ cases:
  - valid payload → correct DataEvent (verify all fields: EventID, EventType, Source, Capability, Operation, EntityID, IdempotencyKey, Backend, Data)
  - invalid JSON → error
  - empty body → error
  - payload with known event types produces correct Operation derivation
  - payload with unknown event type produces empty Operation
- **TestInterfaceCompliance**: compile-time `var _ ability.WebhookConverter = (*Webhook)(nil)` check

### Event type mapping (TestConvert_EventType)

4 cases: `bookmark.created` → Operation `"created"`, `bookmark.updated` → `"updated"`, `bookmark.archived` → `"archived"`, `bookmark.deleted` → `"deleted"`.

## Configuration

### docs/reference/config.yaml and flowbot.yaml

```yaml
vendors:
  karakeep:
    endpoint: "..."
    api_key: "..."
    webhook_token: ""       # new: Bearer token for webhook verification
```

### Provider config reader (pkg/providers/karakeep/karakeep.go)

Following the `GetWebhookSecret()` pattern in `pkg/providers/example/example.go`:

```go
const WebhookTokenKey = "webhook_token"

// GetWebhookToken reads the webhook Bearer token from the karakeep provider config.
func GetWebhookToken() string {
    tok, err := providers.GetConfig(ID, WebhookTokenKey)
    if err != nil {
        return ""
    }
    return tok.String()
}
```

## Error Handling

| Scenario | HTTP Status | Error |
|----------|------------|-------|
| Unknown webhook path | 404 | (returned by WebhookHandler) |
| Webhook token not configured (empty) | 401 | `ErrUnauthorized` (fail closed — returns error at verification time) |
| Missing/wrong Authorization header | 401 | `ErrUnauthorized` |
| Invalid JSON body | 400 | `ErrInvalidArgument` |
| Valid webhook | 202 | (accepted, async emit) |
