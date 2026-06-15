# Server Package

HTTP server with Fiber v3, routing, protocol handlers, and fx dependency injection.

## Structure

```
internal/server/
├── fx.go         # fx App builder, module/provide/invoke wiring
├── server.go     # Server bootstrap (fx.New) and start
├── router.go     # Route registration, platform callbacks
├── http.go       # HTTP server setup and lifecycle
├── error.go      # Error mapping (types.Err* → HTTP status codes)
├── init.go       # Server initialization logic
├── database.go   # Store adapter, database migration
├── platform.go   # Platform driver lifecycle management
├── module.go     # Module registration lifecycle
├── providers.go   # Provider client wiring
├── notify.go     # Notify provider registration
├── chat.go       # Chat message routing
├── pipeline.go   # Pipeline orchestration handlers
├── webhook.go    # Inbound webhook processing
├── hub.go        # Hub app lifecycle handlers
├── agent.go      # AI agent handlers
├── capability.go # Capability registration and health
├── event.go      # Event bus and handler wiring
├── homelab.go    # Homelab scanner lifecycle
├── media.go      # Media file serving
├── message.go    # Message pipeline
├── message_direct.go # Direct message handling
├── chatagent_handler.go # Chat agent handler entry point
├── chatagent_http.go    # Chat agent HTTP routes
├── func.go       # Function lifecycle hooks
├── globals.go    # Global state/singletons
├── reexec.go     # Self-reexec for upgrades
├── registration.go # Platform/ability registration
├── chatagent/     # Chat agent service (run, session, skill, sink, prompt cache, stream coalescer, context usage)
├── swagger.go    # OpenAPI/Swagger docs
├── page_data.go  # Page data serving
└── *_test.go     # Co-located tests
```

## fx Dependency Injection

The server uses `go.uber.org/fx` for dependency injection. The app graph is built in `fx.go`:

```go
fx.New(
    fx.Provide(fx.Annotate(/* constructors */)),
    fx.Invoke(/* initializer functions */),
)
```

- **Provide**: Registers constructors that create singletons (adapters, drivers, clients)
- **Invoke**: Calls initialization functions that need the DI graph (module registration, platform startup)
- New modules wire via `fx.Invoke` in `fx.go`
- New providers wire via `fx.Invoke` in `providers.go` and `notify.go`

## Rules

- Never block in handlers — use goroutines for long ops
- Use protocol helpers (`protocol.NewFailedResponse`, `protocol.NewSuccessResponse`) for structured responses; map `types.Err*` sentinels in `error.go` to HTTP status codes
- Always validate inputs before processing
- Platform drivers are wired via `fx.Provide` and started via `fx.Invoke` in `platform.go`
- Module handlers are registered via `fx.Invoke` in `module.go`

## Routing

- Business routes: `/service/{capability}/*` (injected by modules via `Webservice()`)
- Management routes: `/hub/*` (hub lifecycle, health checks)
- Static assets: `/static/*` (embedded `webassets.FS`)
- Platform callbacks: `/platform/{platform}` (Slack, Discord, Tailchat webhooks)
- Swagger docs: `/swagger/*`

## Testing

```bash
go test ./internal/server/...
```
