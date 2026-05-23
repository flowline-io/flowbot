# Example Ability & Provider Reference Implementation

## Purpose

Provide reference implementations for developers creating new abilities and providers in the flowbot project. The example code demonstrates the full wiring chain: **Module → Ability → Provider → External API**.

## Reference Chain

```
internal/modules/example/      Module: Init, Hub Descriptor, EventSource, Fiber routes
        │  ability.Invoke()
pkg/ability/example/            Ability: Service interface, Descriptor, Conformance, Webhook/Polling
        │  calls provider
pkg/providers/example/          Provider: HTTP client + OAuth → httpbin.org
        │
https://httpbin.org             External API for demonstration
```

## Directory Structure

```
pkg/providers/example/
├── example.go            # Provider struct, GetClient, NewExample, CRUD, OAuth
├── types.go              # Request/response/webhook-payload types
└── example_test.go       # TDD: client methods, config, errors (table-driven)

pkg/ability/example/
├── interface.go          # Service interface (7 methods)
├── descriptor.go         # Descriptor, RegisterService, operation constants, IsMutation
├── webhook.go            # WebhookConverter implementation
├── poller.go             # PollingResource implementation
├── conformance.go        # RunExampleConformance entry point
├── example/              # Provider adapter (mirrors karakeep/, kanboard/, miniflux/)
│   ├── adapter.go        # Implements Service interface, wraps providers/example client
│   ├── adapter_test.go   # TDD: adapter method unit tests with mock client
│   └── conformance_test.go  # BDD: adapter conformance suite
├── descriptor_test.go    # TDD: descriptor registration, operation constant assertions
├── webhook_test.go       # TDD: signature verification, payload conversion
├── poller_test.go        # TDD: single poll cycle, cursor, diff/hash detection
└── conformance_test.go   # TDD: conformance helper self-assertions

internal/modules/example/
├── module.go             # moduleHandler + module.Base, Register(), Init(), Rules(), Webservice()
├── webservice.go         # REST: GET/POST/DELETE /service/example/* rule definitions
├── webhook.go            # Webhook callback rule: EventSourceManager.WebhookHandler()
├── module_test.go        # TDD: handler unit tests with mock ability
└── module_suite_test.go  # BDD: full-chain integration (Ginkgo + Gomega)
```

## Layer Design

### Provider: `pkg/providers/example/`

Demonstrates the complete provider contract using httpbin.org as the external API.

**Constants:**

- `ID = "example"` — provider identifier
- `EndpointKey = "endpoint"` — config key, defaults to `https://httpbin.org`
- `TokenKey = "token"` — optional auth token config key

**`Example` struct** wraps a `resty.Client`. Constructed via `GetClient()` (reads config from `providers.GetConfig`) and `NewExample(endpoint, token string)`.

**API methods** map to httpbin endpoints:

| Method                                           | Endpoint               | Demonstrates                |
| ------------------------------------------------ | ---------------------- | --------------------------- |
| `Get(path string) (*Response, error)`            | `GET /get`             | Read operation with context |
| `Post(path string, data any) (*Response, error)` | `POST /post`           | Write operation with body   |
| `Put(path string, data any) (*Response, error)`  | `PUT /put`             | Update operation            |
| `Delete(path string) (*Response, error)`         | `DELETE /delete`       | Delete operation            |
| `GetStatus(code int) (*Response, error)`         | `GET /status/{code}`   | Error response handling     |
| `GetWithDelay(seconds int) (*Response, error)`   | `GET /delay/{seconds}` | Timeout/deadline handling   |

**OAuth interface** (implemented for reference):

- `GetAuthorizeURL() string` — returns constructed OAuth authorize URL
- `GetAccessToken(ctx fiber.Ctx) (types.KV, error)` — code exchange flow

**Types:**

- `Response` — mirrors httpbin response JSON
- `WebhookPayload` — example webhook event structure
- Config key constants

**Tests (TDD):** table-driven, >= 3 cases per table. Cover happy path, error scenarios, config reading, OAuth methods. Use `httptest.Server` to mock httpbin.

### Ability: `pkg/ability/example/`

Demonstrates the ability adapter pattern wrapping the example provider.

**Service interface** (7 methods):

- `GetItem(ctx, id string) (*ability.Host, error)` — query single
- `ListItems(ctx, q *ListQuery) (*ability.ListResult[ability.Host], error)` — list with pagination
- `CreateItem(ctx, url string) (*ability.Host, error)` — create/mutation
- `UpdateItem(ctx, id string, data map[string]any) error` — update
- `DeleteItem(ctx, id string) error` — delete
- `HealthCheck(ctx) (bool, error)` — health/status
- `ListRawEvents(ctx, cursor string) ([]any, string, error)` — for polling data source

**Concrete Adapter** (`example/adapter.go`):

- `Adapter` struct wrapping `*example.Example` provider client
- `client` interface (local, unexported) for testability
- `func New() abilityexample.Service` — primary constructor, calls `providers.GetClient()`
- `func NewWithClient(client client) abilityexample.Service` — test constructor
- Implements all 7 methods of `Service` interface
- Each method: checks ctx.Err(), validates inputs, calls client method, wraps errors with `types.WrapError`
- Follows exact pattern: `bookmark/karakeep/adapter.go`, `kanban/kanboard/adapter.go`, `reader/miniflux/adapter.go`

**Descriptor** (`descriptor.go`):

- Capability constant: `CapabilityExample hub.CapabilityType = "example"`
- Operation constants for each service method, with mutation verbs properly marked (create, update, delete → `IsMutation`)
- `Descriptor(backend, app string, svc Service) hub.Descriptor` — builds hub descriptor with auth scopes per operation
- `RegisterService(backend, app string, svc Service) error` — calls Descriptor then registers each operation as an `ability.Invoker`, parsing params from `map[string]any`, calling the Service, and returning `*ability.InvokeResult`

**WebhookConverter** (`webhook.go`):

- `ExampleWebhook` struct implementing `ability.WebhookConverter`
- `WebhookPath() string` — returns webhook URL path
- `VerifySignature(headers map[string]string, body []byte) error` — HMAC-SHA256 verification
- `Convert(body []byte, headers map[string]string) ([]types.DataEvent, error)` — transforms webhook payload into DataEvent records

**PollingResource** (`poller.go`):

- `ExamplePoller` struct implementing `ability.PollingResource`
- `ResourceName() string` — unique resource identifier
- `DefaultInterval() time.Duration` — 60s polling interval
- `DiffKey(item any) string` — key for change detection
- `ContentHash(item any) string` — SHA256 hash for content comparison
- `CursorField() string` — field name for cursor-based pagination
- `List(ctx, cursor) (PollResult, error)` — batch fetch with cursor support

**Conformance** (`conformance.go`):

- `ExampleConfig` struct — controls mock backend behavior per test case
- `ExampleServiceFactory func(t, cfg) Service` — factory for creating test services
- `RunExampleConformance(t, factory)` — runs all conformance subtests (success, empty list, timeout, provider error, invalid input)

**Tests (TDD):**

- `descriptor_test.go` — operation constant assertions, RegisterService behavior, Descriptor structure
- `webhook_test.go` — signature verification, valid/invalid payload conversion
- `poller_test.go` — single poll cycle, cursor advancement, diff detection
- `conformance_test.go` — conformance helper self-assertions

### Module: `internal/modules/example/`

Demonstrates the full startup wiring following the standard module contract.

**`module.go`** — Core module structure:

- Package-level `const Name = "example"`; `var handler moduleHandler`
- `func Register() { module.Register(Name, &handler) }`
- `type moduleHandler struct { module.Base; initialized bool }`
- `type configType struct { Enabled bool }` with JSON tags
- `func (moduleHandler) Init(jsonconf json.RawMessage) error`:
  1. Parse config; if `!Config.Enabled`, return nil (graceful disable)
  2. Create adapter: `adapter := abilityexample.New()` (provider creation inside adapter)
  3. Register ability: `abilityexample.RegisterService("example", app, adapter)`
  4. Set initialized = true
- `func (moduleHandler) IsReady() bool { return handler.initialized }`
- `func (moduleHandler) Rules() []any` — returns webserviceRules, webhookRules
- `func (moduleHandler) Webservice(app *fiber.App)` — `module.Webservice(app, Name, webserviceRules)`

**`webservice.go`** — REST rule definitions:

- Rule structs with `Path`, `Method`, `Handler`, `Scopes` following `module.FiberRule` pattern
- `GET /service/example/get` → `ability.Invoke(ctx, abilityexample.Cap, "get", params)`
- `GET /service/example/list` → list with pagination
- `POST /service/example/create` → body → params → invoke
- `DELETE /service/example/delete` → mutation invoke
- `GET /service/example/health` → health check
- Each handler demonstrates: params extraction, invoke call, error classification, response marshaling

**`webhook.go`** — Webhook rule:

- `POST /service/example/webhook/*` → delegates to `esm.WebhookHandler()`
- Demonstrates how Fiber wildcard routes connect to EventSourceManager via module rules

**Tests:**

- **TDD** (`module_test.go`) — table-driven handler unit tests with mocked ability registry (registering a test invoker that returns canned results). Tests: handler routing, params parsing, error responses, pagination.
- **BDD** (`module_suite_test.go`) — Ginkgo v2 integration tests:
  - `Describe("Example module")`
  - `Context("when calling GET /service/example/get")` — validates response structure
  - `Context("when calling POST /service/example/create")` — validates mutation + cache invalidation
  - `Context("webhook delivery")` — validates signature verification + event emission
  - Uses `SynchronizedBeforeSuite` for test setup, mock httpbin via `httptest`

## Testing Standards

### TDD (Unit Tests)

- `*_test.go` co-located with source files
- Table-driven: `for _, tt := range tests { t.Run(tt.name, ...) }`
- Minimum 3 cases per table
- Happy path first, followed by error cases
- Use `t.Parallel()` where independent
- Provider tests use `httptest.Server` to mock httpbin
- Ability tests use mock Service implementations

### BDD (Integration Tests)

- Ginkgo v2 + Gomega
- `Describe` / `Context` / `It` structure
- `SynchronizedBeforeSuite` for shared setup
- `GinkgoParallelProcess()` for parallel execution
- Tests the full chain: HTTP request → handler → ability.Invoke → provider mock → response

## Coverage Checklist

### Provider Layer

- [x] Constants (ID, config keys)
- [x] GetClient() + NewXxx() constructor pattern
- [x] CRUD API methods (Get, Post, Put, Delete)
- [x] Error handling (GetStatus with non-200 codes)
- [x] Timeout handling (GetWithDelay)
- [x] OAuth interface (GetAuthorizeURL, GetAccessToken)
- [x] Webhook payload types
- [x] Response types
- [x] TDD unit tests

### Ability Layer

- [x] Service interface with query + mutation methods
- [x] Adapter implementation (example/adapter.go) wrapping provider client
- [x] Adapter TDD unit tests (example/adapter_test.go)
- [x] Adapter BDD conformance tests (example/conformance_test.go)
- [x] Descriptor + RegisterService
- [x] Operation constants
- [x] IsMutation marking for write operations
- [x] ability.Invoke delegation pattern
- [x] WebhookConverter implementation
- [x] PollingResource implementation
- [x] Conformance test entry (RunExampleConformance)
- [x] TDD unit tests for descriptor, webhook, poller, conformance

### Module Layer

- [x] `moduleHandler` struct embedding `module.Base`
- [x] `Register()` → `module.Register(Name, &handler)`
- [x] `Init(jsonconf) error` with `configType{Enabled bool}` and graceful disable
- [x] `Rules() []any` returning rule slices
- [x] `Webservice(app)` using `module.Webservice(app, Name, rules)`
- [x] Provider → Adapter → ability wiring inside `Init()`
- [x] EventSource registration (webhook + polling)
- [x] REST rule definitions (GET, POST, DELETE) in `webservice.go`
- [x] Webhook callback rule in `webhook.go`
- [x] TDD unit tests for handlers
- [x] BDD integration tests for full chain

## Conventions

- godoc comments: "what" and "why", not "how"
- Errors wrapped with `types.Errorf` using appropriate codes
- Imports: stdlib → third-party → internal
- No panics outside initialization
- No hardcoded credentials
- Use `sonic` for JSON, not `encoding/json`
- Follow existing file naming: lower-case, no underscores except `_test`
