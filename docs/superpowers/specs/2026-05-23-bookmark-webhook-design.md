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

| File                                            | Purpose                                                  |
| ----------------------------------------------- | -------------------------------------------------------- |
| `pkg/ability/bookmark/karakeep/webhook.go`      | `Webhook` struct implementing `capability.WebhookConverter` |
| `pkg/ability/bookmark/karakeep/webhook_test.go` | Table-driven unit tests                                  |

### Modified files

| File                                  | Change                                                                                                      |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `pkg/types/event.go`                  | Add `EventBookmarkUpdated`, `EventBookmarkDeleted` event constants                                          |
| `pkg/ability/event_source_manager.go` | Add `SetEventSourceManager` / `GetEventSourceManager` global accessor (follows `pool.go` pattern)           |
| `internal/server/fx.go`               | Move `initPipeline` before `handleModules` in `fx.Invoke` so the manager is set before module `Bootstrap()` |
| `internal/server/pipeline.go`         | Call `capability.SetEventSourceManager(srcMgr)` after creating the manager                                     |
| `pkg/providers/karakeep/types.go`     | Add `WebhookPayload` struct                                                                                 |
| `pkg/providers/karakeep/karakeep.go`  | Add `WebhookTokenKey` constant and `GetWebhookToken()` config reader                                        |
| `internal/modules/bookmark/module.go` | Register WebhookConverter with EventSourceManager in `Bootstrap()`                                          |
| `docs/reference/config.yaml`          | Add `webhook_token` to `vendors.karakeep`                                                                   |
| `flowbot.yaml`                        | Add `webhook_token` to karakeep config                                                                      |

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

| DataEvent field  | Source                                                                            |
| ---------------- | --------------------------------------------------------------------------------- |
| `EventID`        | `types.Id()`                                                                      |
| `EventType`      | `payload.EventType`                                                               |
| `Source`         | `"karakeep_webhook"`                                                              |
| `Capability`     | `"bookmark"`                                                                      |
| `Operation`      | Inferred from event type: `"created"`, `"updated"`, `"archived"`, or `"deleted"`  |
| `EntityID`       | `payload.Data.Id`                                                                 |
| `IdempotencyKey` | `payload.Data.Id`                                                                 |
| `Backend`        | `"karakeep"`                                                                      |
| `Data`           | `types.KV{"bookmark": toBookmark(payload.Data), "event_type": payload.EventType}` |

## WebhookConverter Implementation

### File skeleton (webhook.go)

```go
package karakeep

import (
    "strings"

    "github.com/bytedance/sonic"

    "github.com/flowline-io/flowbot/pkg/capability"
    provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
    "github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements capability.WebhookConverter for Karakeep.
// It validates Bearer token auth and converts Karakeep webhook payloads.
type Webhook struct {
    getToken func() string
}

// NewWebhook creates a Webhook that reads the Bearer token from provider config
// lazily at verification time (following the example webhook pattern).
func NewWebhook() *Webhook {
    return &Webhook{
        getToken: provider.GetWebhookToken,
    }
}

// Compile-time interface check.
var _ capability.WebhookConverter = (*Webhook)(nil)
```

### WebhookPath

Returns `"karakeep/events"` — the URL path segment under `/webhook/provider/`. The full URL is `/webhook/provider/karakeep/events`.

```go
func (*Webhook) WebhookPath() string {
    return "karakeep/events"
}
```

### VerifySignature

Bearer token validation against the configured `webhook_token`:

```go
func (w *Webhook) VerifySignature(headers map[string]string, _ []byte) error {
    token := w.getToken()
    if token == "" {
        return types.Errorf(types.ErrUnauthorized, "webhook token not configured")
    }
    auth, ok := headers["Authorization"]
    if !ok {
        return types.Errorf(types.ErrUnauthorized, "missing Authorization header")
    }
    const prefix = "Bearer "
    if !strings.HasPrefix(auth, prefix) {
        return types.Errorf(types.ErrUnauthorized, "invalid Authorization header format")
    }
    provided := auth[len(prefix):]
    if provided != token {
        return types.Errorf(types.ErrUnauthorized, "invalid Bearer token")
    }
    return nil
}
```

Note: the `body` parameter is accepted to satisfy the interface but unused (Bearer auth doesn't sign the body).

### Convert

Follows the example's Convert pattern. Signal: `Convert(body []byte, _ map[string]string) ([]types.DataEvent, error)`. The `headers` parameter is accepted to satisfy the interface but unused (Bearer auth is handled in VerifySignature).

```go
func (*Webhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
    var payload provider.WebhookPayload
    if err := sonic.Unmarshal(body, &payload); err != nil {
        return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
    }

    op := strings.TrimPrefix(payload.EventType, "bookmark.")

    ev := types.DataEvent{
        EventID:        types.Id(),
        EventType:      payload.EventType,
        Source:         "karakeep_webhook",
        Capability:     "bookmark",
        Operation:      op,
        EntityID:       payload.Data.Id,
        IdempotencyKey: payload.Data.Id,
        Backend:        "karakeep",
        Data:           types.KV{"bookmark": toBookmark(payload.Data), "event_type": payload.EventType},
    }
    return []types.DataEvent{ev}, nil
}
```

The `toBookmark` helper is defined in `pkg/ability/bookmark/karakeep/adapter.go` — no import needed (same package).

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

`internal/server/pipeline.go` calls `capability.SetEventSourceManager(srcMgr)` after `NewEventSourceManager(...)`.

## Registration Flow

The bookmark module handler must add a `Bootstrap()` method. `module.Base` provides a default no-op; override it:

```go
func (moduleHandler) Bootstrap() error {
    if !Config.Enabled {
        return nil
    }
    mgr := capability.GetEventSourceManager()
    if mgr == nil {
        return fmt.Errorf("bookmark: event source manager not initialized")
    }
    mgr.RegisterWebhook(karakeep.NewWebhook())
    flog.Info("bookmark: registered karakeep webhook on /webhook/provider/karakeep/events")
    return nil
}
```

The `karakeep` import refers to `"github.com/flowline-io/flowbot/pkg/capability/bookmark/karakeep"`. The `ability` import refers to `"github.com/flowline-io/flowbot/pkg/capability"`.

## Testing

### Unit tests (webhook_test.go)

Table-driven tests following the `pkg/ability/example/example/webhook_test.go` pattern. File: `pkg/ability/bookmark/karakeep/webhook_test.go`, package `karakeep`.

**Test injection pattern.** The `getToken` closure is injected directly in the struct literal, matching the example's `getSecret` injection:

```go
w := &Webhook{getToken: func() string { return tt.token }}
```

This avoids calling `karakeep.GetWebhookToken()` (which reads config from disk) in tests.

**Test cases:**

- **TestWebhookPath**: 3+ cases verifying path is always `"karakeep/events"` (structure matches example: "returns path", "consistent path", "always the same")
- **TestVerifySignature**: 5+ cases:
  | name | token | Authorization header | wantErr |
  |------|-------|---------------------|---------|
  | valid Bearer token | `"secret"` | `"Bearer secret"` | false |
  | missing Authorization header | `"secret"` | _(absent)_ | true |
  | wrong token | `"secret"` | `"Bearer wrong"` | true |
  | missing Bearer prefix | `"secret"` | `"secret"` | true |
  | empty configured token returns error | `""` | _(any)_ | true |
- **TestConvert**: 5+ cases:
  | name | body | wantErr | asserts |
  |------|------|---------|---------|
  | valid payload | `{"event_type":"bookmark.created","data":{"id":"b-1",...}}` | false | all DataEvent fields per output table |
  | invalid JSON | `{invalid` | true | — |
  | empty body | `{}` | false | event has empty fields (matching example: no error, zero-value fields) |
  | partial payload | `{"event_type":"bookmark.updated"}` | false | Operation derived, other fields zero-valued |
  | unknown event type | `{"event_type":"bookmark.unknown","data":{"id":"b-1"}}` | false | Operation = `"unknown"` (TrimPrefix pass-through) |
- **TestConvert_EventType**: 4 cases verifying Operation derivation:
  | event_type | Operation |
  |------------|-----------|
  | `bookmark.created` | `"created"` |
  | `bookmark.updated` | `"updated"` |
  | `bookmark.archived` | `"archived"` |
  | `bookmark.deleted` | `"deleted"` |
- **TestInterfaceCompliance**: compile-time `var _ capability.WebhookConverter = (*Webhook)(nil)` check

## Configuration

### docs/reference/config.yaml and flowbot.yaml

```yaml
vendors:
  karakeep:
    endpoint: "..."
    api_key: "..."
    webhook_token: "" # new: Bearer token for webhook verification
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

| Scenario                             | HTTP Status | Error                                                                |
| ------------------------------------ | ----------- | -------------------------------------------------------------------- |
| Unknown webhook path                 | 404         | (returned by WebhookHandler)                                         |
| Webhook token not configured (empty) | 401         | `ErrUnauthorized` (fail closed — returns error at verification time) |
| Missing/wrong Authorization header   | 401         | `ErrUnauthorized`                                                    |
| Invalid JSON body                    | 400         | `ErrInvalidArgument`                                                 |
| Valid webhook                        | 202         | (accepted, async emit)                                               |
