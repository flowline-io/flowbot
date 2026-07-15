# Server Package

HTTP server with Fiber v3, routing, protocol handlers, and fx dependency injection.

## Structure

```
internal/server/
├── fx.go         # fx App Modules; Provide drivers/adapters; Invoke lifecycle handlers
├── server.go     # Server bootstrap (fx.New) and start
├── router.go     # Route registration, platform callbacks
├── http.go       # HTTP server setup and lifecycle
├── error.go      # Error mapping (types.Err* → HTTP status codes)
├── init.go       # Server initialization logic
├── database.go   # Store adapter, database migration
├── platform.go   # handlePlatform: WebSocket start / Shutdown lifecycle
├── module.go     # handleModules: module Init/register after fx wiring
├── providers.go  # OAuth provider registration (github/slack/dropbox)
├── notify.go     # Notify channel registration (fx.Invoke)
├── chat.go       # Chat message routing
├── pipeline.go   # Pipeline orchestration handlers
├── webhook.go    # Inbound webhook processing
├── hub.go        # Hub app lifecycle / health handlers
├── agent.go      # Legacy instruct/collect agent actions
├── capability.go # Capability hub adapter registration (initCapabilityHub)
├── event.go      # Event bus and handler wiring
├── homelab.go    # Homelab scanner lifecycle
├── media.go      # Media backend registration (fs/minio)
├── message.go    # Message pipeline
├── message_direct.go # Direct message handling
├── chatagent_handler.go # Chat agent handler entry point
├── chatagent_http*.go   # Chat agent HTTP routes (sessions, messages, …)
├── agent_ability.go     # Wires chatagent runner into capability agent
├── chatagent_scheduler.go # Chatagent TaskScheduler lifecycle
├── func.go       # Cache store + Fiber struct validator setup
├── globals.go    # Global state/singletons
├── reexec.go     # shell.Register via ReexecModules
├── registration.go # Platform/ability registration
├── chatagent/    # Chat agent service (session, skills, permissions, memory, subagents, scheduler, sink, stream, …)
├── swagger.go    # OpenAPI/Swagger docs (build tag: swagger)
├── page_data.go  # Expired page_data cleanup
└── *_test.go     # Co-located tests
```

## fx Dependency Injection

The server uses `go.uber.org/fx` for dependency injection. The app graph is assembled in `fx.go`:

```go
fx.New(
    fx.Provide(/* constructors */),
    fx.Invoke(/* lifecycle / init functions */),
)
```

- **Provide**: Constructors for singletons (e.g. `slack.NewDriver` in `fx.go`)
- **Invoke**: Startup hooks such as `handleModules`, `handlePlatform` (defined in `module.go` / `platform.go`, invoked from `fx.go`)
- **New modules**: add `fx.Invoke(<pkg>.Register)` in `internal/modules/fx.go` (imported via `modules.Modules`)
- **OAuth providers**: `fx.Invoke` in `providers.go`
- **Notify channels**: `fx.Invoke` in `notify.go`
- Tailchat driver is constructed inside the controller (not `fx.Provide`); Discord package exists but is not wired into the server graph yet

## Rules

- Never block in handlers — use goroutines for long ops
- Use protocol helpers (`protocol.NewFailedResponse`, `protocol.NewSuccessResponse`) for structured responses; map `types.Err*` sentinels in `error.go` to HTTP status codes
- Always validate inputs before processing
- Platform WebSocket lifecycle runs via `handlePlatform` (`platform.go`) on fx Lifecycle
- Module Init/register runs via `handleModules` (`module.go`) after module `Register()` from `internal/modules/fx.go`

## Routing

- Business routes: `/service/{module}/*` (modules register via `Webservice()` → `pkg/route`)
- Management routes: `/hub/*` (hub lifecycle, health checks)
- Chat agent API: `/chatagent/*`
- Static assets: `/static/*` (embedded `webassets.SubFS`)
- Platform callbacks: `/platform/{platform}` (wired: Slack, Tailchat)
- Swagger docs: `/swagger/*` (when built with `-tags swagger`)
- Also: `/oauth/:provider/:flag`, `/form`, `/agent`, `/metrics`, `/livez`, `/readyz`

## Testing

```bash
go test ./internal/server/...
```
