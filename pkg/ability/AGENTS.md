# Ability Guide

Capability abstraction layer that decouples modules from providers. Each capability defines a `Service` interface; provider adapters implement it, and invoker functions route `ability.Invoke` calls through the adapter.

## Structure

```
ability/
├── ability.go               # Shared domain types (Bookmark, ForgeIssue, Host, InvokeResult, ...)
├── invoke.go                # Invoker registry, Invoke(), SetEventEmitter()
├── params.go                # Param extractors (RequiredString, PageRequestFromParams, ...)
├── page.go                  # PageRequest, PageInfo, ListResult[T]
├── cursor.go                # Cursor encoding with per-adapter secrets
├── operations.go            # Operation constants + IsMutation() + Operations map
├── event_source.go          # WebhookConverter interface, PollingResource interface, PollResult
├── poll_scheduler.go        # Poll scheduling infrastructure
├── polling_state.go         # Per-resource poll state with cursor persistence
├── pool.go                  # Adapter pool for health/availability
├── event_source_manager.go  # Orchestrates pollers and webhook hooks
├── webhook_hook.go          # HTTP handler for webhook delivery
├── conformance/             # Shared conformance suites + helpers (one file per capability)
│   ├── conformance.go       # CanceledContext, CursorSecret, TestTime, RequireListResult, ...
│   ├── forge.go             # ForgeConfig + RunForgeConformance()
│   ├── kanban.go            # KanbanConfig + RunKanbanConformance()
│   ├── bookmark.go          # BookmarkConfig + RunBookmarkConformance()
│   └── reader.go            # ReaderConfig + RunReaderConformance()
├── example/                 # Reference capability — follow this for new capabilities
│   ├── interface.go         # Service interface + query types
│   ├── descriptor.go        # Descriptor(), RegisterService(), per-operation invoke*()
│   ├── poller.go            # PollingResource implementation (optional)
│   ├── conformance.go       # Self-contained conformance suite (Config + ServiceFactory)
│   └── example/             # Reference provider adapter — follow this for new backends
│       ├── adapter.go       # Adapter struct implementing Service, New() / NewWithClient()
│       ├── webhook.go       # WebhookConverter implementation (optional)
│       ├── adapter_test.go  # TDD unit tests (table-driven, mock client)
│       ├── webhook_test.go  # Webhook signature verification tests
│       └── conformance_test.go  # Wires factory to the conformance suite
├── <capability>/            # Per-capability: forge, kanban, bookmark, reader, notify, ...
│   ├── interface.go         # Service interface
│   ├── descriptor.go        # Descriptor(), RegisterService(), invoke*() per operation
│   ├── params.go            # Capability-specific param helpers (optional)
│   └── <backend>/           # Per-provider adapter
│       ├── adapter.go       # Adapter struct implementing <capability>.Service
│       ├── webhook.go       # WebhookConverter (optional, if provider sends webhooks)
│       ├── adapter_test.go
│       ├── webhook_test.go
│       └── conformance_test.go
```

## Key Patterns

### Service Interface

Every capability declares a `Service` interface in `interface.go`. Modules call `ability.Invoke(CapType, OpName, params)` — they never import providers or call adapter methods directly.

```go
// pkg/ability/bookmark/interface.go
type ListQuery struct {
    Page       ability.PageRequest
    Archived   *bool
    Favourited *bool
    Tags       []string
}

type Service interface {
    List(ctx context.Context, q *ListQuery) (*ability.ListResult[ability.Bookmark], error)
    Get(ctx context.Context, id string) (*ability.Bookmark, error)
    Create(ctx context.Context, url string) (*ability.Bookmark, error)
    Archive(ctx context.Context, id string) (bool, error)
    // ...
}
```

### Descriptor + Invoker Registration

`descriptor.go` wires the `Service` to the hub and invoker registry. Each operation gets an `invoke*` closure that extracts params, calls the service method, and wraps the result in `*ability.InvokeResult`.

```go
// pkg/ability/bookmark/descriptor.go
func Descriptor(backend, app string, svc Service) hub.Descriptor {
    return hub.Descriptor{
        Type:        hub.CapBookmark,
        Backend:     backend,
        App:         app,
        Description: "Bookmark capability",
        Instance:    svc,
        Healthy:     svc != nil,
        Operations: []hub.Operation{
            {Name: ability.OpBookmarkList, Description: "List bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead}},
            {Name: ability.OpBookmarkCreate, Description: "Create a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
            // ...
        },
    }
}

func RegisterService(backend, app string, svc Service) error {
    if svc == nil {
        return types.Errorf(types.ErrInvalidArgument, "bookmark service is required")
    }
    if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
        return err
    }
    for _, item := range []struct{ operation string; invoker ability.Invoker }{
        {ability.OpBookmarkList, invokeList(svc)},
        {ability.OpBookmarkCreate, invokeCreate(svc)},
        // ...
    } {
        if err := ability.RegisterInvoker(hub.CapBookmark, item.operation, item.invoker); err != nil {
            return err
        }
    }
    return nil
}
```

- Use `ability.OpXxx` constants from `operations.go` for operation names — never define local duplicates.
- Param extraction uses `ability.RequiredString()` / `ability.PageRequestFromParams()` / `ability.IntParam()` — never raw type assertions.
- Mutation operations may populate `InvokeResult.Events` with `[]ability.EventRef` to emit data events, or `InvokeResult.Resource` with `*ability.ResourceMeta` for audit/idempotency tracking:

```go
// Emitting events (preferred for mutation operations)
return &ability.InvokeResult{
    Data: item,
    Text: "bookmark created: " + item.URL,
    Events: []ability.EventRef{{
        EventType: types.EventBookmarkCreated,
        EntityID:  item.ID,
    }},
}, nil

// Audit/idempotency tracking (alternative)
return &ability.InvokeResult{
    Data: item,
    Resource: &ability.ResourceMeta{
        EntityID: item.ID,
        App:      backend,
    },
}, nil
```

- List operations commonly use a `listInvokeResult` helper to ensure non-nil `Items` and `Page`:

```go
func listInvokeResult(operation string, result *ability.ListResult[ability.Bookmark]) *ability.InvokeResult {
    if result == nil {
        result = &ability.ListResult[ability.Bookmark]{Items: []*ability.Bookmark{}, Page: &ability.PageInfo{}}
    }
    return &ability.InvokeResult{Operation: operation, Data: result.Items, Page: result.Page}
}
```

### Adapter Pattern (Provider -> Service)

Each backend lives in `pkg/ability/<capability>/<backend>/adapter.go`. The adapter wraps the provider client and implements the capability's `Service` interface.

```go
// pkg/ability/bookmark/karakeep/adapter.go
type client interface {
    GetAllBookmarks(query *provider.BookmarksQuery) (*provider.BookmarksResponse, error)
    GetBookmark(id string) (*provider.Bookmark, error)
    CreateBookmark(url string) (*provider.Bookmark, error)
    // ... only methods the adapter actually uses
}

type Adapter struct {
    client       client
    cursorSecret []byte
    now          func() time.Time
}

func New() bm.Service {
    return NewWithClient(provider.GetClient())
}

func NewWithClient(c client) bm.Service {
    return &Adapter{
        client:       c,
        cursorSecret: defaultCursorSecret,
        now:          time.Now,
    }
}
```

- `New()` reads config from YAML via the provider client; `NewWithClient()` accepts an injected client for testing.
- The `client` interface is a **subset** of the provider's exported type — only the methods the adapter actually uses.
- Include `var _ Service = (*Adapter)(nil)` compile-time interface check in the backend test file.
- Include `var _ client = (*fakeClient)(nil)` compile-time check in the conformance test file.
- Adapters **never** call `hub`, `pipeline`, or emit `DataEvent` directly. They only call provider methods and map results to ability domain types.
- Wrap provider errors with `types.WrapError(types.ErrProvider, "context", err)`.
- Always check `ctx.Err()` at the top of each method and return `types.ErrTimeout` when canceled.
- Expose `SetCursorSecret(secret []byte)` for tests that need deterministic cursor encoding.

### WebhookConverter (Optional)

When a provider sends webhooks, implement `ability.WebhookConverter` in the backend directory:

```go
// pkg/ability/bookmark/karakeep/webhook.go
type Webhook struct {
    getToken func() string
}

// Compile-time interface check.
var _ ability.WebhookConverter = (*Webhook)(nil)

func NewWebhook() *Webhook { ... }
func (*Webhook) WebhookPath() string { return "karakeep/events" }
func (w *Webhook) VerifySignature(headers map[string]string, body []byte) error { ... }
func (*Webhook) Convert(body []byte, headers map[string]string) ([]types.DataEvent, error) { ... }
```

- Always include `var _ ability.WebhookConverter = (*Webhook)(nil)` for compile-time safety.
- `VerifySignature` validates HMAC, Bearer token, or other provider-specific schemes.
- `Convert` parses the raw body with `sonic.Unmarshal` and returns `[]types.DataEvent` — each event must include a unique `EventID` and `IdempotencyKey`.
- Register the webhook via `ability.EventSourceManager.RegisterWebhook()`.

### PollingResource (Optional)

When a provider lacks webhooks, implement `ability.PollingResource`:

```go
// pkg/ability/example/poller.go
type ExamplePoller struct { svc Service; secret []byte }

func (*ExamplePoller) ResourceName() string { ... }
func (*ExamplePoller) DefaultInterval() time.Duration { ... }
func (*ExamplePoller) DiffKey(item any) string { ... }
func (*ExamplePoller) ContentHash(item any) string { ... }
func (*ExamplePoller) CursorField() string { ... }
func (p *ExamplePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) { ... }
```

- `Service` should expose a `ListRawEvents` method that the poller delegates to.
- Register via `ability.EventSourceManager.RegisterPollingResource()`.

### Conformance Tests

Conformance suites live in `pkg/ability/conformance/<capability>.go`. Each defines a `Config` struct, a `ServiceFactory` type, and a `RunXxxConformance` function that tests every Service method across success, timeout, provider-error, and invalid-input scenarios.

```go
// pkg/ability/conformance/forge.go
type ForgeConfig struct { User *ability.ForgeUser; UserErr error /* ... one field per method + error */ }
type ForgeServiceFactory func(t *testing.T, cfg ForgeConfig) forge.Service

func RunForgeConformance(t *testing.T, factory ForgeServiceFactory) {
    t.Run("get user success", func(t *testing.T) { /* ... */ })
    t.Run("get user timeout", func(t *testing.T) { /* ... */ })
    // ... one subtest per method x scenario
}
```

Backend adapters wire their fake client in `<backend>/conformance_test.go`:

```go
conformance.RunForgeConformance(t, func(_ *testing.T, cfg conformance.ForgeConfig) forge.Service {
    c := &fakeClient{user: cfgToSDKUser(cfg.User), userErr: cfg.UserErr /* ... */}
    a := NewWithClient(c).(*Adapter)
    a.cursorSecret = conformance.CursorSecret
    a.now = conformance.TestTime
    return a
})
```

- Reference `pkg/ability/example/example/conformance_test.go` for the full adapter wiring pattern.
- Use `conformance.CursorSecret` / `conformance.TestTime` for deterministic cursor tests.
- Use `conformance.RequireXxxError` helpers in suite implementations.

### Descriptor Tests

Each capability must include `descriptor_test.go`. Reference `pkg/ability/example/descriptor_test.go` for the canonical pattern.

- **Nil service**: `Descriptor()` returns `Healthy: false`; `RegisterService()` returns error.
- **Non-nil service**: `Descriptor()` returns `Healthy: true`, correct `CapType`/`Backend`/`App`/`Description`/`Instance`.
- **Operations list**: every `ability.OpXxx` constant appears in `Descriptor().Operations`; assert exact count via `assert.Len`.
- Use a mock `Service` (not an adapter) — self-contained, no provider dependency.
- Use `hub.CapXxx` / `ability.OpXxx` constants, never hardcode strings.

### Operation Constants

All operation names are defined in `pkg/ability/operations.go` as package-level constants keyed by capability:

```go
// pkg/ability/operations.go
const (
    OpBookmarkList       = "list"
    OpBookmarkCreate     = "create"
    // ...
)

const (
    OpForgeGetUser        = "get_user"
    OpForgeGetRepo        = "get_repo"
    OpForgeListIssues     = "list_issues"
    // ...
)
```

The `Operations` map provides key-to-value lookup. The `IsMutation()` helper detects write operations by verb. Register new operation constants and add entries to `Operations` for each new capability.

Descriptors reference these via `ability.OpXxx` — never redefine locally.

## Rules

- **Reference implementation**: When creating or modifying a capability or adapter, reference `pkg/ability/example/` for file structure, naming, and code style.
- **Modules never import providers** — use `ability.Invoke(CapType, OpName, params)` in module webservice handlers.
- **Adapters never call hub/pipeline/emit DataEvent** — they return domain types and errors; the invoker layer handles event emission via `Events` / `Resource` fields.
- **Adapters never return provider-private types** — map everything to `ability.*` domain types or `types.KV`.
- **Deserialization** uses `sonic.Unmarshal`, never `encoding/json`.
- **Error wrapping**: use `types.WrapError(types.ErrProvider, "context", err)` for provider failures, `types.Errorf(types.ErrInvalidArgument, ...)` for validation, return `types.ErrNotFound` with `types.Errorf` or `types.WrapError` for missing entities.
- **Context deadline**: check `ctx.Err()` at the top of every adapter method; wrap as `types.ErrTimeout`.
- **Pagination**: use `ability.PageRequest` / `ability.PageInfo` / `ability.ListResult[T]`. Cursor-based pagination uses `ability.EncodeCursor` / `ability.DecodeCursor` with per-adapter secrets.
- **Operations map** in `operations.go` must include every operation for the capability — used by audit and UI rendering.
- **Operation constants** go in `operations.go` as `ability.OpXxx`; descriptors reference them, never define their own.
- **Compile-time checks**: `var _ Service = (*Adapter)(nil)` in backend test file, `var _ WebhookConverter = (*Webhook)(nil)` in webhook file, `var _ client = (*fakeClient)(nil)` in conformance test.
- **Conformance suites** go in `pkg/ability/conformance/<capability>.go` with `Config`, `ServiceFactory`, and `RunXxxConformance`.
- **Capability URLs**: module webservice routes go under `/service/{capability}`, hub management under `/hub/*`.

## Testing

- **Adapter unit tests** (`adapter_test.go`) use table-driven tests with a mock provider client. Cover happy path, empty inputs, error propagation, and context cancellation. Minimum 3 cases per table.
- **Webhook tests** (`webhook_test.go`) verify signature/token validation and payload conversion edge cases.
- **Conformance suites** in `pkg/ability/conformance/` cover every service method across success, timeout, provider error, and invalid-input scenarios.
- **Conformance adapter tests** (`conformance_test.go`) wire the backend's fake client to the shared conformance suite via `ServiceFactory`.
- **Descriptor tests** (`descriptor_test.go`) verify `Descriptor()` output structure and `RegisterService()` coverage.
- Mock HTTP with `httptest` for providers that make HTTP calls — never hit real services in unit tests.
- BDD specs for end-to-end capability flows go in `tests/specs/`.
