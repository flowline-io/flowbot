# Plugin System Design

**Date:** 2026-06-05
**Status:** Approved
**Approach:** Adapter Bridge (Approach A)

## Overview

A plugin system for flowbot that allows third-party extensibility, independent deployment of new capabilities, and eventually a plugin marketplace. Plugins can act as modules, ability backends, and providers (full stack). Two execution environments are supported: gRPC (subprocess) and Wasm/WASI (sandbox via wazero), built in parallel.

## Goals

- Allow external developers to extend flowbot without modifying the core codebase
- Support full hot-reload (load/unload/update without restarting flowbot)
- Provide a minimal host API (config, logging, KV storage, HTTP client, event emission)
- Go SDK for plugin development (TinyGo for Wasm targets)
- Hybrid distribution: local filesystem for dev, OCI registry for production, Git for community plugins

## Non-Goals

- Multi-language SDKs beyond Go (future work)
- Plugin-to-plugin communication (plugins communicate through flowbot's existing event system)
- Plugin UI/dashboard (future work)

## Architecture

### Three-Layer Runner Architecture

```
┌──────────────────────────────────────────────────┐
│                  Plugin Manager                   │
│    ┌────────────┐  ┌────────────┐                │
│    │  gRPC      │  │  Wasm/WASI  │                │
│    │  Runner    │  │  Runner     │                │
│    │ (子进程)   │  │ (沙箱)      │                │
│    └────────────┘  └────────────┘                │
│                                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │           Plugin Adapter Layer               │ │
│  │ GrpcModuleAdapter / WasmModuleAdapter / ...  │ │
│  └──────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
         ↓ registers into
┌──────────────┐  ┌───────────────────┐  ┌──────────┐
│ Hub Registry │  │ Ability Registry  │  │  Routes  │
└──────────────┘  └───────────────────┘  └──────────┘
```

| Runner    | Startup | Isolation    | Performance     | Use Case                              |
|-----------|---------|--------------|-----------------|---------------------------------------|
| gRPC      | ~50ms   | Process-level| gRPC overhead   | Third-party adapters, complex plugins |
| Wasm/WASI | ~5ms    | Sandbox      | Serialization   | Filters, rule engines, lightweight    |

### Package Layout

```
pkg/plugin/
├── manager.go          # PluginManager: discovery, lifecycle, hot-reload
├── manifest.go         # plugin.yaml parsing and validation
├── runner.go           # Runner interface (common contract)
├── hostapi.go          # Host API interface (services exposed to plugins)
├── grpc/
│   ├── runner.go       # gRPC runner using hashicorp/go-plugin
│   ├── host.go         # HostService gRPC server (flowbot side)
│   ├── plugin.go       # PluginService gRPC server (plugin side, in SDK)
│   └── proto/          # Protobuf definitions (shared between host and plugin)
├── wasm/
│   ├── runner.go       # Wasm runner using github.com/wazero/wazero
│   ├── host.go         # Host function imports (WASI + custom)
│   └── memory.go       # Memory read/write helpers
├── adapter/
│   ├── module.go       # PluginModuleAdapter → module.Handler
│   ├── ability.go      # PluginAbilityAdapter → ability.Service
│   └── provider.go     # PluginProviderAdapter → providers.OAuthProvider
├── source/
│   ├── source.go       # Source interface (Load, Watch, Close)
│   ├── local.go        # Filesystem source
│   ├── oci.go          # OCI registry source
│   └── git.go          # Git repo source
└── sdk/
    ├── module.go       # Go SDK: module plugin interface
    ├── ability.go      # Go SDK: ability plugin interface
    ├── provider.go     # Go SDK: provider plugin interface
    ├── host.go         # Go SDK: host API client
    └── serve.go        # Go SDK: go-plugin.Serve() boilerplate
```

## Plugin Manifest

Each plugin declares its metadata, runtime, capabilities, and permissions in `plugin.yaml`:

```yaml
name: my-plugin
version: "1.2.0"
description: "Example plugin for flowbot"
author: "developer@example.com"
runtime: grpc | wasm

provides:
  module: true
  abilities:
    - capability: bookmark
      operations: [list, get, create]
  provider:
    name: my-service
    oauth: true

grpc:
  binary: ./plugin-server
  args: ["--port", "0"]

wasm:
  module: ./plugin.wasm
  permissions:
    http:
      - host: "api.example.com"
    filesystem:
      - path: "/tmp/flowbot/my-plugin"
        mode: "readwrite"
    memory:
      max: "64MB"
    execution:
      timeout: "30s"

config_schema:
  type: object
  properties:
    api_key:
      type: string
      description: "API key for the service"
  required: [api_key]
```

The manifest is validated at load time. The `provides` section determines which adapters are created. A plugin can provide any combination of module, abilities, and provider.

**Plugin identity:** The `name` field in `plugin.yaml` is untrusted. The PluginManager derives the actual plugin identity from:

- **Local source:** Subdirectory name (e.g., `./plugins/my-plugin/` → `my-plugin`)
- **OCI source:** `org/repo` from the image reference (e.g., `ghcr.io/org/repo:tag` → `org/repo`)
- **Git source:** `org/repo` from the Git URL

The derived identity is used for all registration (`module.Register(id, ...)`), KV prefixing (`plugin:<id>:`), and API endpoints. The manifest's `name` field is preserved as a display label only. Duplicate identities are rejected at load time.

The `config_schema` field (JSON Schema) is used by the PluginManager to validate the per-plugin configuration from `flowbot.yaml` before passing it to `runner.Start()`. If validation fails, the plugin is not loaded and an error is logged. This allows plugin developers to declare required configuration fields and get early validation feedback.

## Runner Interface

```go
// Runner is the common contract for plugin execution environments.
type Runner interface {
    Load(ctx context.Context, manifest *Manifest) (*PluginInfo, error)
    Start(ctx context.Context, config json.RawMessage) error
    Stop(ctx context.Context) error
    Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error)
    Health(ctx context.Context) (*HealthStatus, error)
}
```

The `Call` method uses a single string-based dispatch (`"command"`, `"form"`, `"rules"`, `"ability_call"`, etc.) with JSON params and JSON results. Each runner implementation maps these to its native invocation:

- **gRPC Runner:** Maps function name to the corresponding `PluginService` RPC, serializing params to protobuf
- **Wasm Runner:** Maps function name to the corresponding wasm export (e.g., `"command"` → `command(ptr, size) → i64`), using the JSON-over-memory protocol

This abstraction keeps adapters runner-agnostic — they never know whether a plugin runs via gRPC or Wasm.

```go

## gRPC Runner (hashicorp/go-plugin)

Uses `github.com/hashicorp/go-plugin`, a battle-tested framework for Go process-based plugins. It provides: bidirectional gRPC over local connections, handshake protocol, process lifecycle management, stdio multiplexing for logs, and automatic subprocess cleanup (no orphans).

### Communication Model

`go-plugin` uses gRPC over a local connection (Unix socket or localhost TCP). Flowbot acts as the **host**, the plugin binary as the **plugin process**. The framework handles connection negotiation, health pings, and graceful shutdown.

```
┌──────────────┐          ┌─────────────────────┐
│   flowbot     │ gRPC     │   Plugin Process     │
│              │◄────────►│                      │
│ go-plugin    │ local    │ go-plugin client      │
│ Server       │ socket   │ + Plugin impl         │
│ + Host API   │          │                      │
└──────────────┘          └─────────────────────┘
```

### Protobuf Service Definitions

`go-plugin` uses a handshake + gRPC service pattern. The host exposes `HostService` and the plugin exposes `PluginService`:

```protobuf
// Plugin process exposes this service to flowbot
service PluginService {
    rpc Init(InitRequest) returns (InitResponse);
    rpc Bootstrap(go_plugin.KitchenSink.Empty) returns (go_plugin.KitchenSink.Empty);
    rpc Command(CommandRequest) returns (CommandResponse);
    rpc Form(FormRequest) returns (FormResponse);
    rpc Rules(go_plugin.KitchenSink.Empty) returns (RulesResponse);
    rpc Help(go_plugin.KitchenSink.Empty) returns (HelpResponse);
    rpc IsReady(go_plugin.KitchenSink.Empty) returns (IsReadyResponse);
    rpc AbilityCall(AbilityCallRequest) returns (AbilityCallResponse);
    rpc WebhookConvert(WebhookRequest) returns (WebhookResponse);
    rpc OAuthCallback(OAuthRequest) returns (OAuthResponse);
}

// Flowbot exposes this service to the plugin (Host API)
service HostService {
    rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
    rpc Log(LogRequest) returns (go_plugin.KitchenSink.Empty);
    rpc KVGet(KVGetRequest) returns (KVGetResponse);
    rpc KVSet(KVSetRequest) returns (go_plugin.KitchenSink.Empty);
    rpc KVDelete(KVDeleteRequest) returns (go_plugin.KitchenSink.Empty);
    rpc HTTPRequest(HTTPRequest) returns (HTTPResponse);
    rpc EmitEvent(EmitEventRequest) returns (go_plugin.KitchenSink.Empty);
}
```

### Plugin Lifecycle (go-plugin managed)

`go-plugin` handles subprocess lifecycle:
- **Launch:** `plugin.Serve()` in the plugin binary, `plugin.ClientConfig.Cmd` in the host
- **Handshake:** Both sides verify protocol version via a magic cookie over stdio before serving gRPC
- **Health:** Built-in bidirectional pings. If the plugin process exits unexpectedly, `go-plugin` signals the host
- **Cleanup:** `client.Kill()` on the host side. On Linux, `SysProcAttr.Pdeathsig = syscall.SIGKILL` ensures child processes die when the parent exits abnormally (OOM, kill -9). On non-Linux platforms (macOS, Windows), this field is omitted via `//go:build linux` build tags; go-plugin's built-in heartbeat mechanism serves as fallback — when stdin closes or bidirectional pings timeout, the plugin process self-terminates.
- **Stdio:** Plugin's stdout/stderr are forwarded to host's logger via `plugin.ClientConfig.Stderr`/`Stdout` hooks

### Hot-reload

1. Create new go-plugin client with updated binary
2. Wait for new client handshake + `IsReady() == true`
3. Drain in-flight calls on old client (graceful stop with 30s drain timeout)
4. Swap adapter references to new runner
5. Kill old client via `client.Kill()`

**Statelessness requirement:** Plugins must be stateless. In-process state (memory caches, rate limit counters, WebSocket connections) is lost during hot-reload. Any persistent state must use the Host API `KVGet`/`KVSet` functions, which survive hot-reload because they store data in flowbot's namespaced KV store (`plugin:<name>:*` prefix). The SDK enforces this by not exposing any stateful primitives to plugins.

## Wasm Runner

### Runtime

Pure Go via `github.com/wazero/wazero`. Zero CGO. Each plugin gets its own `wazero.Runtime` for isolation.

```go
type WasmRunner struct {
    runtime    wazero.Runtime
    compiled   wazero.CompiledModule
    instance   wazeroapi.Module
    hostAPI    *HostAPIBindings
    functions  map[string]wazeroapi.Function
    memLimit   uint32
    timeout    time.Duration
}
```

### Memory Protocol

**Decision: JSON over shared linear memory (not the WIT Component Model ABI).**

The WIT document at the end of this spec serves as the **canonical interface specification** for plugin developers, documenting function signatures and types. However, the actual wire format used at runtime is **JSON serialized into Wasm linear memory** via the `alloc`/`free` export convention. This is simpler to implement in a pure wazero runtime (no Component Model support needed) and avoids the complexity of `wit-bindgen` tooling for each target language.

The WIT file is the "source of truth" for the interface contract; the JSON-over-memory encoding is the runtime transport. If wazero gains full Component Model support in the future, the runtime layer can be upgraded transparently.

```
┌────────────────────────────────────────────┐
│  Wasm Linear Memory                        │
│                                            │
│  [alloc function] → returns ptr to buffer  │
│  [host writes JSON to ptr, passes ptr+size]│
│  [plugin reads, processes, writes result]  │
│  [host reads result from returned ptr+size]│
└────────────────────────────────────────────┘
```

**Required wasm exports:**

- `alloc(size: i32) -> i32` — allocate buffer in wasm memory
- `free(ptr: i32)` — release buffer (allocator tracks size internally)
- `init(ptr: i32, size: i32) -> i64` — returns `(ptr << 32) | size` of result
- `command(ptr: i32, size: i32) -> i64`
- `form(ptr: i32, size: i32) -> i64`
- `rules() -> i64`
- `help() -> i64`
- `is_ready() -> i32`
- `bootstrap() -> i64`
- `ability_call(ptr: i32, size: i32) -> i64`
- `webhook_convert(ptr: i32, size: i32) -> i64`

All functions returning `i64` encode `(result_ptr << 32) | result_size`. All Wasm responses use a standard JSON envelope:

```json
{"error": null, "data": ...}     // success
{"error": "message", "data": null}  // failure
```

This avoids a separate `last_error()` export and its associated extra call overhead. The host reads a single result buffer and checks the `error` field.

### Host Function Imports

Registered under the `flowbot` namespace:

```go
func (h *HostAPIBindings) getConfig(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize, outPtr uint32) uint32
func (h *HostAPIBindings) log(ctx context.Context, mod wazeroapi.Module, level, msgPtr, msgSize uint32)
func (h *HostAPIBindings) kvGet(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize uint32) uint64
func (h *HostAPIBindings) kvSet(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize, valPtr, valSize uint32) uint32
func (h *HostAPIBindings) kvDelete(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize uint32) uint32
func (h *HostAPIBindings) httpRequest(ctx context.Context, mod wazeroapi.Module, reqPtr, reqSize uint32) uint64
func (h *HostAPIBindings) emitEvent(ctx context.Context, mod wazeroapi.Module, eventPtr, eventSize uint32) uint32
```

### WASI Integration

Use wazero's built-in WASI preview1 support, restricted by manifest permissions:

- **Filesystem:** Only declared paths mounted via `wazero.WithFSConfig()`
- **Network:** Blocked by default. HTTP access goes through the `httpRequest` host function (enforces allowlist)
- **Clock:** Allowed (needed for timeouts)
- **Random:** Allowed (needed for UUIDs, nonces)

### Hot-reload

1. Compile new `.wasm` module
2. Instantiate new module instance with fresh memory
3. Call `init()` on new instance and verify `is_ready() == true`
4. Swap the runner's `instance` reference atomically (new calls go to the new instance)
5. Wait for in-flight calls on old instance to drain, using a `sync.WaitGroup` with the configured `drain_timeout` (default 30s). Each `Runner.Call()` adds a WaitGroup delta; the drain loop waits for the counter to reach zero.
6. Close old instance via `wazeroapi.Module.Close(ctx)` — wazero handles resource cleanup

No subprocess management needed. Wazero instances are lightweight Go objects.

**Drain guarantee:** Without drain, abruptly closing an in-flight instance would cause forced request failure. The WaitGroup drain ensures all existing calls complete before the old instance is destroyed, matching the gRPC runner's drain behavior.

### Instance Pooling

For high-throughput plugins, an optional pool of pre-instantiated wasm modules:

```go
type WasmInstancePool struct {
    pool     sync.Pool       // pool of wazeroapi.Module
    inflight sync.WaitGroup  // track in-flight executions per instance
    manifest *Manifest
    compiled wazero.CompiledModule
}
```

**Concurrency scheduling:** `wazeroapi.Module` linear memory is not concurrency-safe. On each `Runner.Call()`, the runner borrows an instance from the pool, executes the call synchronously, and returns the instance to the pool. Each pool instance is used by at most one goroutine at a time.

**Pool limits:** The manifest's `wasm.pool.max_instances` (default 4) caps the number of pre-instantiated modules. When the pool is exhausted, callers wait up to `wasm.pool.wait_timeout` (default 5s) for an instance to be returned. If the timeout expires, the call fails fast with an `ErrPluginBusy` error (mapped to HTTP 503), preventing Goroutine pile-up under high concurrency with slow plugin operations.

```yaml
wasm:
  module: ./plugin.wasm
  pool:
    max_instances: 4
    wait_timeout: 5s
  permissions:
    # ...
```

**Memory cleanup:** After each call completes, the host calls the wasm `free(ptr)` export for any buffers allocated during that call (the allocator tracks sizes internally). This prevents linear memory from growing unboundedly. The pool may also enforce a max memory watermark — if an instance exceeds it, the instance is discarded (closed) rather than returned to the pool, and a fresh instance takes its place.

## Adapter Layer

### Module Adapter

```go
// PluginModuleAdapter implements module.Handler by delegating to a Runner.
type PluginModuleAdapter struct {
    module.Base
    runner   Runner
    manifest *Manifest
    ready    atomic.Bool
}

func (a *PluginModuleAdapter) Init(jsonconf json.RawMessage) error {
    return a.runner.Start(context.Background(), jsonconf)
}

func (a *PluginModuleAdapter) Command(ctx types.Context, content any) (types.MsgPayload, error) {
    params := marshalCall("command", ctx, content)
    result, err := a.runner.Call(ctx, "command", params)
    if err != nil {
        return types.MsgPayload{}, err
    }
    return unmarshalPayload(result)
}
// Form, Rules, Help, Bootstrap follow the same pattern

// Webservice: plugins cannot directly mount Fiber routes. Instead,
// plugin-declared webservice rules are proxied through a generic
// /service/{plugin-name}/* route in flowbot. The adapter forwards
// requests to the plugin via Runner.Call("webservice", serializedRequest).
```

### Ability Adapter

```go
// PluginAbilityAdapter registers Invoker closures for declared operations.
// Rather than implementing a per-capability Service interface (which has
// capability-specific method signatures), the adapter registers generic
// Invoker functions into ability.RegisterInvoker(). Each Invoker serializes
// the operation name and params, calls Runner.Call("ability_call", ...),
// and deserializes the result.
type PluginAbilityAdapter struct {
    runner     Runner
    capability hub.CapabilityType
    operations []string
}

func (a *PluginAbilityAdapter) Register() error {
    for _, op := range a.operations {
        ability.RegisterInvoker(a.capability, op, a.makeInvoker(op))
    }
    return nil
}

func (a *PluginAbilityAdapter) makeInvoker(op string) ability.Invoker {
    return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
        payload := marshal(map[string]any{"operation": op, "params": params})
        result, err := a.runner.Call(ctx, "ability_call", payload)
        if err != nil {
            return nil, err
        }
        return unmarshalInvokeResult(result)
    }
}
```

Only declared operations are registered. Undeclared operations remain unregistered and return `types.ErrNotFound` from `ability.Invoke()`.

### Provider Adapter

```go
// PluginProviderAdapter implements provider OAuth and webhook interfaces.
type PluginProviderAdapter struct {
    runner Runner
    name   string
}
```

### Registration Flow

When `PluginManager.Load()` processes a manifest:

1. `manifest.provides.module == true` → Create `PluginModuleAdapter` → `module.Register(name, adapter)`
2. `manifest.provides.abilities != nil` → For each ability: create `PluginAbilityAdapter` → `ability.RegisterInvoker(cap, op, adapterFunc)` → `hub.Default.Register(descriptor)`
3. `manifest.provides.provider != nil` → Create `PluginProviderAdapter` → `providers.RegisterOAuthProvider(name, factory)` if `oauth: true`

### Unregistration (for hot-reload)

When the manifest's `provides` section changes (capabilities added/removed):

1. `module.Unregister(name)`
2. `ability.UnregisterInvoker(cap, ops...)`
3. `hub.Default.Unregister(cap)`
4. `providers.UnregisterOAuthProvider(name)`
5. `runner.Stop(ctx)` — drain + kill

**When `provides` is unchanged (same capabilities, same operations):** Skip registry manipulation entirely. Perform an **atomic swap** of the adapter's underlying `Runner` pointer. This eliminates the "service vacuum" window between unregister and register where incoming requests would receive 404 errors. Use `atomic.Pointer[Runner]` within each adapter.

**Hot-reload flow (complete):**

```
1. Validate new manifest + config (fail → abort, old plugin keeps running)
2. Create new runner, Load() + Start()
3. Wait for new runner IsReady() == true
4. If provides changed:
   → Unregister old adapters, register new ones
5. If provides unchanged:
   → Atomic swap adapter runner pointer (gap-free)
6. Drain old runner (30s timeout) → Stop
```


## Plugin Manager

```go
// PluginManager orchestrates plugin discovery, loading, lifecycle, and hot-reload.
type PluginManager struct {
    plugins   map[string]*PluginInstance
    runners   map[string]Runner
    sources   []Source
    mu        sync.RWMutex
    logger    zerolog.Logger
}

type PluginInstance struct {
    Manifest  *Manifest
    Runner    Runner
    Adapters  []any
    State     PluginState     // loading, running, stopping, error
    StartedAt time.Time
    LastError error
}

type PluginState string
const (
    StateLoading  PluginState = "loading"
    StateRunning  PluginState = "running"
    StateStopping PluginState = "stopping"
    StateError    PluginState = "error"
)
```

### Lifecycle

**Startup:**

1. `PluginManager.Init(config)` — parse plugin sources from `flowbot.yaml`
2. For each source: discover plugins (scan dirs, pull OCI, clone git)
3. For each discovered plugin:
   - Parse + validate manifest
   - Select runner (grpc or wasm) based on `manifest.runtime`
   - `runner.Load(manifest)`
   - Create adapters based on `manifest.provides`
   - Register adapters into flowbot registries
   - `runner.Start(config)`
   - Mark as running

**Hot-reload (API call or file watcher):**

**Guarantee: validate-then-swap with automatic rollback.** If any step fails before the swap, the old plugin keeps running and an alert is logged. Specifically:

1. Load new manifest + binary
2. Validate new `config_schema` against existing `flowbot.yaml` config — if schema added required fields not present, **abort** (old plugin keeps running, alert logged)
3. Create new runner, `Load()` + `Start()` with existing config
4. Wait for new runner `IsReady() == true`
5. If provides changed: unregister old, register new. If unchanged: atomic swap.
6. Drain old runner (30s timeout) → Stop
7. Swap `PluginInstance` references

Step 2 prevents the "unloaded old, failed to load new" scenario. A failed hot-reload never leaves the system without the plugin.

**Unload (API call):**

1. Unregister all adapters
2. `runner.Stop()` with drain
3. Remove from plugins map

### API Endpoints

```
POST   /hub/plugins/load         — Load a plugin from a source
DELETE /hub/plugins/:name         — Unload a plugin
POST   /hub/plugins/:name/reload — Hot-reload a plugin
GET    /hub/plugins              — List all loaded plugins + status
GET    /hub/plugins/:name/health — Plugin health details
```

## Host API

```go
// HostAPI defines the services available to plugins.
type HostAPI interface {
    GetConfig(ctx context.Context, key string) (string, error)
    Log(ctx context.Context, level string, msg string, fields map[string]string)
    KVGet(ctx context.Context, key string) ([]byte, error)
    KVSet(ctx context.Context, key string, value []byte) error
    KVDelete(ctx context.Context, key string) error
    HTTPRequest(ctx context.Context, req *HostHTTPRequest) (*HostHTTPResponse, error)
    EmitEvent(ctx context.Context, event types.DataEvent) error
}
```

**KV storage:** Each plugin gets a namespace prefix (`plugin:<name>:`) in the existing KV store. Plugins cannot access other plugins' or the core's KV data.

**HTTP request enforcement:** For Wasm plugins, the `httpRequest` host function checks the manifest's `wasm.permissions.http` allowlist. gRPC plugins rely on network isolation (unix socket only) but the host API enforces the same allowlist for consistency.

## Plugin SDK (Go)

### Module Interface

```go
// Module is the interface for module plugins.
type Module interface {
    Init(config json.RawMessage) error
    Bootstrap() error
    Command(ctx *Context, content any) (*MsgPayload, error)
    Form(ctx *Context, values map[string]string) (*MsgPayload, error)
    Rules() (*Rules, error)
    Help() (map[string][]string, error)
    IsReady() bool
}

type Context struct {
    AuthContext string
    UserID      string
    ChannelID   string
    Platform    string
    Metadata    map[string]string
}
```

### gRPC Entry Point

```go
// Called from the plugin's main(). go-plugin handles handshake,
// process lifecycle, and gRPC transport automatically.
func ServeModule(m Module) error {
    impl := &pluginServer{mod: m}
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: plugin.HandshakeConfig{
            ProtocolVersion:  1,
            MagicCookieKey:   "FLOWBOT_PLUGIN",
            MagicCookieValue: "flowbot-plugin-v1",
        },
        Plugins: map[string]plugin.Plugin{
            "module": &grpcPlugin{impl: impl},
        },
        GRPCServer: plugin.DefaultGRPCServer, // serves on a random port
    })
    return nil
}
```

### Wasm Entry Point (TinyGo)

```go
//go:build wasi

func ExportModule(m Module) {
    globalModule = m
}

//export init
func wasmInit(ptr, size uint32) uint64 { ... }
//export command
func wasmCommand(ptr, size uint32) uint64 { ... }
```

### Example Plugin

```go
package main

import "flowbot.dev/plugin/sdk"

type myPlugin struct { sdk.ModuleBase }

func (p *myPlugin) Command(ctx *sdk.Context, content any) (*sdk.MsgPayload, error) {
    return &sdk.MsgPayload{Text: "Hello from plugin!"}, nil
}

func main() {
    sdk.ServeModule(&myPlugin{})
}
```

## Distribution Sources

```go
// Source discovers and provides plugin artifacts.
type Source interface {
    Discover(ctx context.Context) ([]*Manifest, error)
    Artifact(ctx context.Context, name string) ([]byte, error)
    Watch(ctx context.Context) (<-chan SourceEvent, error)
    Close() error
}
```

**Local source:** Watches a directory (`./plugins/`) using `fsnotify`. Each subdirectory contains `plugin.yaml` + binary.

**OCI source:** Pulls plugin images from a registry. Image format: `plugin.yaml` + `plugin.wasm` (or binary for gRPC) + `config.schema.json`.

**Git source:** Clones repos, checks out tagged versions. `plugin.yaml` at repo root.

## Security Model

| Layer            | gRPC                                    | Wasm                                         |
|------------------|-----------------------------------------|----------------------------------------------|
| Process isolation| Separate OS process                     | wazero sandbox (no direct OS access)          |
| Network          | Unix socket only (no TCP)               | No network; HTTP via host function + allowlist|
| Filesystem       | Chroot or seccomp (optional)            | WASI FS restricted to declared paths          |
| Memory           | OS process limits                       | wazero memory limit from manifest             |
| CPU              | cgroup or nice (optional)               | Per-call timeout from manifest                |
| KV               | Namespaced per plugin                   | Namespaced per plugin                         |
| Config           | Plugin sees only its own config         | Plugin sees only its own config               |

**Permission enforcement:**

- Wasm: enforced at the host function level (every `httpRequest` call checks allowlist)
- gRPC: enforced at the host API server level (same checks, defense in depth)
- Both: plugins cannot access other plugins' KV namespaces (prefix isolation)

## Configuration

In `flowbot.yaml`:

```yaml
plugins:
  enabled: true
  sources:
    - type: local
      path: ./plugins
    - type: oci
      registry: ghcr.io/myorg/flowbot-plugins
      poll_interval: 1h
    - type: git
      repos:
        - url: https://github.com/community/flowbot-plugins.git
          ref: main
          poll_interval: 6h

  config:
    my-plugin:
      api_key: "${env:MY_PLUGIN_API_KEY}"
      debug: true

  hot_reload: true
  drain_timeout: 30s
  max_plugins: 50
```

## WIT Interface Definition

For Wasm plugins, the canonical interface contract:

```wit
package flowbot:plugin@1.0.0;

interface types {
    record context {
        auth-context: string,
        user-id: string,
        channel-id: string,
        platform: string,
        metadata: list<tuple<string, string>>,
    }

    record msg-payload {
        text: string,
    }

    record data-event {
        source: string,
        event-type: string,
        payload: string,
    }

    record http-request {
        method: string,
        url: string,
        headers: list<tuple<string, string>>,
        body: list<u8>,
    }

    record http-response {
        status: u16,
        headers: list<tuple<string, string>>,
        body: list<u8>,
    }
}

interface host {
    use types.{context, data-event, http-request, http-response};

    get-config: func(key: string) -> result<string, string>;
    log: func(level: string, msg: string);
    kv-get: func(key: string) -> result<list<u8>, string>;
    kv-set: func(key: string, value: list<u8>) -> result<_, string>;
    kv-delete: func(key: string) -> result<_, string>;
    http-request: func(req: http-request) -> result<http-response, string>;
    emit-event: func(event: data-event) -> result<_, string>;
}

interface module {
    use types.{context, msg-payload};

    init: func(config: string) -> result<_, string>;
    bootstrap: func() -> result<_, string>;
    command: func(ctx: context, content: string) -> result<string, string>;
    form: func(ctx: context, values: list<tuple<string, string>>) -> result<string, string>;
    rules: func() -> result<string, string>;
    help: func() -> result<string, string>;
    is-ready: func() -> bool;
}

interface ability {
    call: func(operation: string, params: string) -> result<string, string>;
}

interface provider {
    webhook-convert: func(payload: list<u8>) -> result<string, string>;
    oauth-authorize: func(state: string) -> result<string, string>;
    oauth-callback: func(code: string) -> result<string, string>;
}

world flowbot-plugin {
    import host;
    export module;
    export ability;
    export provider;
}
```

## Testing Strategy

### Unit Tests

- `pkg/plugin/manifest_test.go` — manifest parsing, validation, schema checks
- `pkg/plugin/manager_test.go` — lifecycle, hot-reload, error handling
- `pkg/plugin/adapter/*_test.go` — adapter bridging, JSON marshaling, error mapping
- `pkg/plugin/wasm/memory_test.go` — memory read/write, alloc/free
- `pkg/plugin/grpc/proto_test.go` — protobuf serialization

### Integration Tests

- Build a test plugin (Go gRPC + TinyGo Wasm) using the SDK
- Load it via PluginManager, exercise all Handler methods
- Test hot-reload: update binary, verify seamless swap
- Test error cases: plugin crash, timeout, permission violation

### BDD Specs (Ginkgo)

```
Describe("Plugin System")
  Context("gRPC plugin")
    It("loads and responds to commands")
    It("hot-reloads without dropping requests")
    It("recovers from plugin crash")
  Context("Wasm plugin")
    It("loads and responds to commands")
    It("enforces memory limits")
    It("enforces HTTP permission allowlist")
    It("hot-reloads atomically")
  Context("Plugin as ability backend")
    It("registers invokers and serves operations")
  Context("Plugin as provider")
    It("converts webhooks to DataEvents")
```

### Conformance

Extend `pkg/ability/conformance/` with plugin-specific test suites that verify adapter compliance.

## Limitations

### Payload Size

gRPC Protobuf and Wasm linear memory JSON transport are both unsuitable for large payloads (e.g., 50MB video files in webhook conversion). The current design assumes request/response payloads under 1MB. For future file-oriented plugins (media processing, document conversion), the Host API will be extended with either:

- **Shared Virtual File Path:** Plugins receive a filesystem path (within their sandboxed directory) instead of file bytes over the RPC boundary
- **Stream RPC:** gRPC streaming endpoints and Wasm host function callbacks for chunked read/write

Neither is implemented in v1. The `wasm.permissions.filesystem` sandboxed paths and the manifest's `wasm.permissions.memory.max` (default 64MB) act as safety limits until these are added.

## Dependencies

| Dependency | Purpose | Status |
|------------|---------|--------|
| `github.com/hashicorp/go-plugin` | gRPC plugin framework (handshake, lifecycle, process management) | New |
| `github.com/wazero/wazero` | Pure-Go Wasm runtime (zero CGO) | Indirect → Direct |
| `google.golang.org/grpc` | gRPC transport (used by go-plugin internally) | Indirect → Direct |
| `google.golang.org/protobuf` | Protobuf serialization | Indirect → Direct |
| `github.com/fsnotify/fsnotify` | File watching for local source | New |
| `github.com/google/go-containerregistry` | OCI image pulling | New |

## Migration Notes

- Add `Unregister` methods to `module`, `ability`, and `providers` registries where missing
- Promote `github.com/wazero/wazero` from indirect to direct dependency
- Add `github.com/hashicorp/go-plugin` as a new direct dependency
- No changes to existing built-in modules, abilities, or providers required
- Plugin system is opt-in via `plugins.enabled: true` in config
