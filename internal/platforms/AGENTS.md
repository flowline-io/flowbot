# Platforms Guide

Multi-platform chat and messaging integrations. Each platform implements the `protocol.Driver`, `protocol.Adapter`, and `protocol.Action` interfaces to provide bidirectional communication with external chat services.

## Reference Implementation

- When creating or modifying a platform, reference `internal/platforms/slack/` for file structure, naming, and code style — it is the most complete implementation.

## Structure

```
internal/platforms/
├── platforms.go          # Caller dispatch, MessageConvert, PlatformRegister, GetCaller
├── <platform>/
│   ├── driver.go         # Driver struct + NewDriver() → protocol.Driver
│   ├── adapter.go        # Adapter struct → protocol.Adapter (MessageConvert, EventConvert)
│   ├── action.go         # Action struct → protocol.Action (SendMessage, etc.)
│   ├── types.go          # const ID, platform-local types
│   ├── *_test.go         # TDD unit tests (table-driven)
│   └── blockkit.go       # Optional: platform-specific rendering helpers (e.g. Slack Block Kit)
```

## Core Package (`platforms.go`)

- **`Caller`** — bundles an `Action` and `Adapter`. `Caller.Do(req)` dispatches `SendMessage`/`UpdateMessage`/`DeleteMessage` to the platform's `Action` based on `req.Action`.
- **`PlatformRegister(name, caller)`** — persists the platform to the database (idempotent) and stores the `Caller` in the in-memory registry.
- **`GetCaller(name)`** — retrieves a registered platform by name from the in-memory registry.
- **`MessageConvert(data)`** — uses `reflect.TypeOf` to dispatch `types.MsgPayload` variants (Text, Link, Table, Info, Chart, Html, Markdown, Instruct, KV, Form, Empty) to their `protocol.Message` converters.

## Interfaces (defined in `pkg/types/protocol`)

- **`Driver`** — lifecycle and transport: `HttpServer`, `HttpWebhookClient`, `WebSocketClient`, `WebSocketServer`, `Shoutdown`
- **`Adapter`** — converts platform-native payloads into `protocol.Message` / `protocol.Event`
- **`Action`** — platform API (all methods must be implemented):
  - Messaging: `SendMessage`, `UpdateMessage`, `DeleteMessage`
  - User: `GetUserInfo`
  - Channel: `CreateChannel`, `GetChannelInfo`, `GetChannelList`
  - Registration: `RegisterChannels`, `RegisterSlashCommands`
  - Query: `GetLatestEvents`, `GetSupportedActions`, `GetStatus`, `GetVersion`

## Patterns

- **New platform**: create a sub-package under `internal/platforms/<name>/` with `driver.go`, `adapter.go`, `action.go`, `types.go`.
- **`types.go`**: define `const ID = "<name>"` (used for DB registration and route dispatch). Add any platform-specific request/response types here.
- **`driver.go`**: `NewDriver(cfg *config.Type, store store.Adapter) protocol.Driver` initializes the SDK client, calls `platforms.PlatformRegister(ID, &platforms.Caller{Action: …, Adapter: …})`, and returns the `Driver`. Wire the driver via `fx.Provide` in `internal/server/fx.go`.
- **`adapter.go`**: `MessageConvert` typically delegates to `platforms.MessageConvert(data)` for common types; `EventConvert` maps platform-specific webhook/interaction payloads to `protocol.Event`.
- **`action.go`**: implement `SendMessage` (required for messaging); unsupported actions return `protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))`. Additional helpers (Block Kit builders, chart rendering, file upload) may live in platform-local files like `blockkit.go`.
- **Route callbacks**: platform HTTP callbacks are dispatched in `router.go` `platformCallback` by matching the `platform` param against platform `ID` constants. The `Controller` may hold platform-specific driver fields (e.g. `tailchatDriver`) in addition to a generic `driver` for the primary platform.
- **Lifecycle**: `handlePlatform` in `server/platform.go` starts `WebSocketClient` on app startup via `fx.Lifecycle`.

## Rules

- Never import `pkg/providers/*` from `internal/platforms/*`.
- Platform packages are internal — do not expose platform-specific types outside `internal/platforms`.
- Platform-native event conversion belongs in the adapter, not in the server.
- Channel/message routing logic lives in the platform's `action.go`, not in `server/`.
- Always use `protocol.NewFailedResponse(protocol.ErrXxx.New(…))` for errors.
- Use `protocol.NewSuccessResponse(data)` for success responses.
- Register each platform in `internal/server/fx.go` and `internal/server/router.go`.

## Testing

- Each component has `*_test.go` counterpart.
- Table-driven tests with `require`/`assert`.
- Mock platform SDK clients or use `httptest` for HTTP-based platforms.
