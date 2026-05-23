# Bookmark Webhook Support

## Motivation

The bookmark ability currently supports REST CRUD operations through the Karakeep adapter, but has no inbound webhook support. Adding a WebhookConverter allows external services (primarily Karakeep) to push bookmark events into flowbot's event pipeline, enabling real-time reactivity (notifications, pipeline triggers, workflow automation) when bookmarks are created, updated, archived, or deleted.

## Design Overview

Add a Karakeep-specific `WebhookConverter` implementation in `pkg/ability/bookmark/karakeep/`, registered with the `EventSourceManager` during the bookmark module's `Init()`. The converter accepts webhook POSTs at `/webhook/provider/karakeep/events`, authenticates via `Authorization: Bearer <token>`, and transforms Karakeep webhook payloads into `types.DataEvent` records.

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
| `internal/server/pipeline.go` | Call `ability.SetEventSourceManager(srcMgr)` after creating the manager |
| `pkg/providers/karakeep/types.go` | Add `WebhookPayload` struct |
| `pkg/providers/karakeep/karakeep.go` | Add `GetWebhookToken()` config reader |
| `internal/modules/bookmark/module.go` | Register WebhookConverter with EventSourceManager in `Init()` |
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
| `IdempotencyKey` | `payload.Data.Id` |
| `Data` | `types.KV{"bookmark": toBookmark(payload.Data), "event_type": payload.EventType}` |

## WebhookConverter Implementation

### Webhook struct

```go
type Webhook struct {
    token string
}

func NewWebhook(token string) *Webhook {
    return &Webhook{token: token}
}
```

### WebhookPath

Returns `"karakeep/events"` — the URL path segment under `/webhook/provider/`.

### VerifySignature

Bearer token validation against the configured `webhook_token`:

1. If `webhook_token` config is empty → return nil (skip verification)
2. Extract `Authorization` header
3. Validate `Bearer ` prefix
4. Compare token value to configured `webhook_token`
5. Mismatch → `types.Errorf(types.ErrUnauthorized, "...")`

### Convert

1. Parse body as `provider.WebhookPayload` using `sonic.Unmarshal`
2. Invalid JSON → `types.Errorf(types.ErrInvalidArgument, "...")`
3. Build single `DataEvent` with bookmark data
4. Return `[]types.DataEvent{ev}`

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

In `internal/modules/bookmark/module.go` `Init()`:

1. Check `Enabled` flag (existing logic)
2. Call `karakeep.GetWebhookToken()` to get the webhook token from config
3. Create `karakeep.NewWebhook(token)`
4. Call `ability.GetEventSourceManager().RegisterWebhook(webhook)`

## Testing

### Unit tests (webhook_test.go)

Table-driven tests following the `pkg/ability/example/webhook_test.go` pattern:

- **TestWebhookPath**: 3+ cases verifying path is always `"karakeep/events"`
- **TestVerifySignature**: 5+ cases:
  - valid Bearer token → no error
  - missing Authorization header → error
  - wrong token → error
  - missing Bearer prefix → error
  - empty configured token → skip verification (no error)
- **TestConvert**: 4+ cases:
  - valid payload → correct DataEvent
  - invalid JSON → error
  - empty body → error
  - payload with known event types maps correctly

### Event type mapping (TestConvert_EventType)

4 cases: `bookmark.created`, `bookmark.updated`, `bookmark.archived`, `bookmark.deleted`.

## Configuration

### docs/reference/config.yaml and flowbot.yaml

```yaml
vendors:
  karakeep:
    endpoint: "..."
    api_key: "..."
    webhook_token: ""       # new: Bearer token for webhook verification
```

## Error Handling

| Scenario | HTTP Status | Error |
|----------|------------|-------|
| Unknown webhook path | 404 | (returned by WebhookHandler) |
| Missing/wrong Authorization header | 401 | `ErrUnauthorized` |
| Invalid JSON body | 400 | `ErrInvalidArgument` |
| Valid webhook | 202 | (accepted, async emit) |
