# Plugin System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a plugin system supporting gRPC (hashicorp/go-plugin) and Wasm/WASI (wazero) runners, allowing plugins to act as modules, ability backends, and providers with full hot-reload support.

**Architecture:** Adapter Bridge pattern — plugins register into existing flowbot registries via `PluginModuleAdapter`, `PluginAbilityAdapter`, and `PluginProviderAdapter`. The `PluginManager` orchestrates discovery (local/OCI/Git sources), lifecycle (load/unload/hot-reload with validate-then-swap), and hosts a minimal Host API (config, logging, KV, HTTP, events).

**Tech Stack:** Go 1.26+, `hashicorp/go-plugin`, `wazero/wazero`, protobuf, JSON Schema, `fsnotify`, `go-containerregistry`

---

## File Structure

```
pkg/plugin/
├── manifest.go         # Manifest type, YAML parsing, validation (JSON Schema)
├── manifest_test.go    # Manifest unit tests
├── runner.go           # Runner interface, PluginInfo, HealthStatus, PluginState
├── hostapi.go          # HostAPI interface + HostHTTPRequest/Response types
├── manager.go          # PluginManager: discovery, lifecycle, hot-reload
├── manager_test.go     # PluginManager unit tests
├── config.go           # PluginConfig for flowbot.yaml integration
├── grpc/
│   ├── proto/
│   │   ├── plugin.proto   # PluginService and HostService definitions
│   │   ├── plugin.pb.go   # Generated protobuf code
│   │   └── plugin_grpc.pb.go # Generated gRPC code
│   ├── runner.go          # GrpcRunner implementing Runner
│   ├── runner_test.go     # GrpcRunner tests
│   ├── host.go            # HostService gRPC server (Host API side)
│   └── plugin.go          # PluginService gRPC server (SDK side, re-exported)
├── wasm/
│   ├── runner.go          # WasmRunner implementing Runner
│   ├── runner_test.go     # WasmRunner tests
│   ├── host.go            # Host function imports (flowbot namespace)
│   ├── memory.go          # alloc/free helpers, JSON read/write
│   └── memory_test.go     # Memory tests
├── adapter/
│   ├── module.go          # PluginModuleAdapter → module.Handler
│   ├── module_test.go     # Module adapter tests
│   ├── ability.go         # PluginAbilityAdapter → ability.Invoker
│   ├── ability_test.go    # Ability adapter tests
│   ├── provider.go        # PluginProviderAdapter → provider interfaces
│   └── provider_test.go   # Provider adapter tests
├── source/
│   ├── source.go          # Source interface + SourceEvent
│   ├── local.go           # Local filesystem source (fsnotify)
│   ├── local_test.go      # Local source tests
│   ├── oci.go             # OCI registry source
│   ├── oci_test.go        # OCI source tests
│   ├── git.go             # Git repo source
│   └── git_test.go        # Git source tests
└── sdk/
    ├── module.go          # Module interface + ModuleBase
    ├── ability.go         # Ability plugin interface
    ├── provider.go        # Provider plugin interface
    ├── host.go            # Host API client (wraps HostService)
    ├── serve.go           # go-plugin.Serve() entry point
    └── types.go           # Context, MsgPayload, Rules types

pkg/config/
├── config.go              # Add Plugins struct + config_schema

internal/modules/
├── fx.go                  # Add fx wiring for PluginManager

internal/server/
├── hub_plugins.go         # Plugin API endpoints (/hub/plugins/*)
└── hub_plugins_test.go    # Plugin endpoint tests

pkg/module/
├── module.go              # Add Unregister method

pkg/ability/
├── invoke.go              # Add UnregisterInvoker method

pkg/providers/
├── providers.go           # Add UnregisterOAuthProvider method

pkg/hub/
├── registry.go            # Add Unregister method
```

---

## Phase 1: Foundation Types

### Task 1: Manifest type and YAML parsing

**Files:**
- Create: `pkg/plugin/manifest.go`
- Create: `pkg/plugin/manifest_test.go`

- [ ] **Step 1: Define manifest types in `pkg/plugin/manifest.go`**

```go
package plugin

import "encoding/json"

// Manifest is the parsed plugin.yaml configuration.
type Manifest struct {
	Name        string          `json:"name" yaml:"name"`
	Version     string          `json:"version" yaml:"version"`
	Description string          `json:"description" yaml:"description"`
	Author      string          `json:"author" yaml:"author"`
	Runtime     RuntimeKind     `json:"runtime" yaml:"runtime"`
	Provides    Provides        `json:"provides" yaml:"provides"`
	GRPC        *GRPCConfig     `json:"grpc" yaml:"grpc"`
	Wasm        *WasmConfig     `json:"wasm" yaml:"wasm"`
	ConfigSchema json.RawMessage `json:"config_schema" yaml:"config_schema"`
}

// RuntimeKind is the plugin execution environment.
type RuntimeKind string

const (
	RuntimeGRPC RuntimeKind = "grpc"
	RuntimeWasm RuntimeKind = "wasm"
)

// Provides declares what the plugin implements.
type Provides struct {
	Module    bool              `json:"module" yaml:"module"`
	Abilities []AbilityDecl     `json:"abilities" yaml:"abilities"`
	Provider  *ProviderDecl     `json:"provider" yaml:"provider"`
}

// AbilityDecl declares a capability the plugin provides as an ability backend.
type AbilityDecl struct {
	Capability string   `json:"capability" yaml:"capability"`
	Operations []string `json:"operations" yaml:"operations"`
}

// ProviderDecl declares a provider plugin.
type ProviderDecl struct {
	Name  string `json:"name" yaml:"name"`
	OAuth bool   `json:"oauth" yaml:"oauth"`
}

// GRPCConfig is the gRPC runner configuration.
type GRPCConfig struct {
	Binary string   `json:"binary" yaml:"binary"`
	Args   []string `json:"args" yaml:"args"`
}

// WasmConfig is the Wasm runner configuration.
type WasmConfig struct {
	Module      string           `json:"module" yaml:"module"`
	Permissions *WasmPermissions `json:"permissions" yaml:"permissions"`
	Pool        *WasmPoolConfig  `json:"pool" yaml:"pool"`
}

// WasmPermissions defines Wasm sandbox permissions.
type WasmPermissions struct {
	HTTP       []HTTPPermission  `json:"http" yaml:"http"`
	Filesystem []FSPermission    `json:"filesystem" yaml:"filesystem"`
	Memory     *MemoryLimit      `json:"memory" yaml:"memory"`
	Execution  *ExecutionLimit   `json:"execution" yaml:"execution"`
}

// HTTPPermission allowslist entry for HTTP requests from Wasm.
type HTTPPermission struct {
	Host string `json:"host" yaml:"host"`
}

// FSPermission defines filesystem access for Wasm plugins.
type FSPermission struct {
	Path string `json:"path" yaml:"path"`
	Mode string `json:"mode" yaml:"mode"` // "readwrite" or "read"
}

// MemoryLimit contains Wasm memory constraints.
type MemoryLimit struct {
	Max string `json:"max" yaml:"max"` // e.g. "64MB"
}

// ExecutionLimit contains execution constraints.
type ExecutionLimit struct {
	Timeout string `json:"timeout" yaml:"timeout"` // e.g. "30s"
}

// WasmPoolConfig contains the Wasm instance pool configuration.
type WasmPoolConfig struct {
	MaxInstances int    `json:"max_instances" yaml:"max_instances"` // default 4
	WaitTimeout  string `json:"wait_timeout" yaml:"wait_timeout"`   // default "5s"
}

// ParseManifest parses plugin.yaml bytes into a Manifest, validating required fields.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := sonic.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("manifest: missing name")
	}
	if m.Runtime != RuntimeGRPC && m.Runtime != RuntimeWasm {
		return nil, fmt.Errorf("manifest: invalid runtime %q, must be grpc or wasm", m.Runtime)
	}
	if m.Runtime == RuntimeGRPC && m.GRPC == nil {
		return nil, fmt.Errorf("manifest: grpc config required for grpc runtime")
	}
	if m.Runtime == RuntimeWasm && m.Wasm == nil {
		return nil, fmt.Errorf("manifest: wasm config required for wasm runtime")
	}
	return &m, nil
}

// ValidateConfig validates per-plugin config against the manifest's config_schema.
// Returns nil if no schema is defined. Uses github.com/santhosh-tekuri/jsonschema/v6.
func (m *Manifest) ValidateConfig(config json.RawMessage) error {
	if len(m.ConfigSchema) == 0 {
		return nil
	}
	schema, err := jsonschema.CompileString("plugin.yaml", string(m.ConfigSchema))
	if err != nil {
		return fmt.Errorf("manifest config_schema is invalid: %w", err)
	}
	var v any
	if err := sonic.Unmarshal(config, &v); err != nil {
		return fmt.Errorf("plugin config is not valid JSON: %w", err)
	}
	if err := schema.Validate(v); err != nil {
		return fmt.Errorf("plugin config validation failed: %w", err)
	}
	return nil
}
```

Imports needed: `"encoding/json"`, `"fmt"`, `"github.com/bytedance/sonic"`, `"github.com/santhosh-tekuri/jsonschema/v6"`.

- [ ] **Step 2: Write manifest parsing tests in `pkg/plugin/manifest_test.go`**

```go
package plugin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		wantErr string
		wantRT  RuntimeKind
	}{
		{
			name: "valid grpc manifest",
			yaml: `name: test-plugin
version: "1.0.0"
runtime: grpc
grpc:
  binary: ./server`,
			wantRT: RuntimeGRPC,
		},
		{
			name: "valid wasm manifest",
			yaml: `name: test-plugin
version: "1.0.0"
runtime: wasm
wasm:
  module: ./plugin.wasm`,
			wantRT: RuntimeWasm,
		},
		{
			name:    "missing name",
			yaml:    `runtime: grpc`,
			wantErr: "missing name",
		},
		{
			name: "invalid runtime",
			yaml: `name: test
runtime: invalid`,
			wantErr: "invalid runtime",
		},
		{
			name: "grpc without grpc config",
			yaml: `name: test
runtime: grpc`,
			wantErr: "grpc config required",
		},
		{
			name: "wasm without wasm config",
			yaml: `name: test
runtime: wasm`,
			wantErr: "wasm config required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ParseManifest([]byte(tt.yaml))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRT, m.Runtime)
		})
	}
}

func TestManifestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schema  json.RawMessage
		config  json.RawMessage
		wantErr string
	}{
		{
			name:    "no schema passes",
			schema:  nil,
			config:  json.RawMessage(`{"any": true}`),
			wantErr: "",
		},
		{
			name: "valid config passes",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {"api_key": {"type": "string"}},
				"required": ["api_key"]
			}`),
			config:  json.RawMessage(`{"api_key": "secret"}`),
			wantErr: "",
		},
		{
			name: "missing required field fails",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {"api_key": {"type": "string"}},
				"required": ["api_key"]
			}`),
			config:  json.RawMessage(`{}`),
			wantErr: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{ConfigSchema: tt.schema}
			err := m.ValidateConfig(tt.config)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests to verify**

Run: `go test ./pkg/plugin/ -run "TestParseManifest|TestManifestValidateConfig" -v -count=1`
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/plugin/manifest.go pkg/plugin/manifest_test.go
git commit -m "feat(plugin): add Manifest type, YAML parsing, and config schema validation"
```

---

### Task 2: Runner interface and common types

**Files:**
- Create: `pkg/plugin/runner.go`

- [ ] **Step 1: Define Runner interface and related types in `pkg/plugin/runner.go`**

```go
package plugin

import (
	"context"
	"encoding/json"
	"time"
)

// Runner is the common contract for plugin execution environments.
// Each runner (gRPC, Wasm) implements this interface.
type Runner interface {
	Load(ctx context.Context, manifest *Manifest) (*PluginInfo, error)
	Start(ctx context.Context, config json.RawMessage) error
	Stop(ctx context.Context) error
	Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error)
	Health(ctx context.Context) (*HealthStatus, error)
}

// PluginInfo is returned after Load() and describes what the plugin provides.
type PluginInfo struct {
	Name         string
	Version      string
	Provides     Provides
	ConfigSchema json.RawMessage
}

// HealthStatus reports plugin readiness and last error.
type HealthStatus struct {
	Ready     bool
	LastError string
	Uptime    time.Duration
}

// PluginState represents the lifecycle state of a loaded plugin.
type PluginState string

const (
	StateLoading  PluginState = "loading"
	StateRunning  PluginState = "running"
	StateStopping PluginState = "stopping"
	StateError    PluginState = "error"
)
```

- [ ] **Step 2: Commit**

```bash
git add pkg/plugin/runner.go
git commit -m "feat(plugin): add Runner interface and common types (PluginInfo, HealthStatus, PluginState)"
```

---

### Task 3: Host API interface

**Files:**
- Create: `pkg/plugin/hostapi.go`

- [ ] **Step 1: Define HostAPI interface in `pkg/plugin/hostapi.go`**

```go
package plugin

import (
	"context"

	"flowbot.dev/pkg/types"
)

// HostAPI defines the services available to plugins from the host.
type HostAPI interface {
	GetConfig(ctx context.Context, key string) (string, error)
	Log(ctx context.Context, level string, msg string, fields map[string]string)
	KVGet(ctx context.Context, key string) ([]byte, error)
	KVSet(ctx context.Context, key string, value []byte) error
	KVDelete(ctx context.Context, key string) error
	HTTPRequest(ctx context.Context, req *HostHTTPRequest) (*HostHTTPResponse, error)
	EmitEvent(ctx context.Context, event types.DataEvent) error
}

// HostHTTPRequest is an HTTP request from a plugin to the host.
type HostHTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// HostHTTPResponse is an HTTP response from the host to a plugin.
type HostHTTPResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/plugin/hostapi.go
git commit -m "feat(plugin): add HostAPI interface and HTTP request/response types"
```

---

## Phase 2: gRPC Runner (hashicorp/go-plugin)

### Task 4: Protobuf service definitions

**Files:**
- Create: `pkg/plugin/grpc/proto/plugin.proto`

- [ ] **Step 1: Add hashicorp/go-plugin dependency**

Run: `go get github.com/hashicorp/go-plugin`
Expected: dependency added to go.mod.

- [ ] **Step 2: Write protobuf definitions in `pkg/plugin/grpc/proto/plugin.proto`**

```protobuf
syntax = "proto3";

package flowbot.plugin;

option go_package = "flowbot.dev/pkg/plugin/grpc/proto;pb";

import "google.golang.org/protobuf/types/known/emptypb.proto";
import "google.golang.org/protobuf/types/known/struct.proto";

// PluginService is exposed by the plugin binary to flowbot.
service PluginService {
    rpc Init(InitRequest) returns (InitResponse);
    rpc Bootstrap(google.protobuf.Empty) returns (google.protobuf.Empty);
    rpc Command(CommandRequest) returns (CommandResponse);
    rpc Form(FormRequest) returns (FormResponse);
    rpc Rules(google.protobuf.Empty) returns (RulesResponse);
    rpc Help(google.protobuf.Empty) returns (HelpResponse);
    rpc IsReady(google.protobuf.Empty) returns (IsReadyResponse);
    rpc AbilityCall(CallRequest) returns (CallResponse);
    rpc WebhookConvert(WebhookConvertRequest) returns (CallResponse);
    rpc OAuthCallback(OAuthCallbackRequest) returns (CallResponse);
}

// HostService is exposed by flowbot to the plugin binary (Host API).
service HostService {
    rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
    rpc Log(LogRequest) returns (google.protobuf.Empty);
    rpc KVGet(KVGetRequest) returns (KVGetResponse);
    rpc KVSet(KVSetRequest) returns (google.protobuf.Empty);
    rpc KVDelete(KVDeleteRequest) returns (google.protobuf.Empty);
    rpc HTTPRequest(HTTPCallRequest) returns (HTTPCallResponse);
    rpc EmitEvent(EmitEventRequest) returns (google.protobuf.Empty);
}

message InitRequest {
    string config = 1; // JSON-encoded config
}

message InitResponse {}

message CommandRequest {
    string context = 1;  // JSON-encoded types.Context
    string content = 2;  // JSON-encoded content
}

message CommandResponse {
    string payload = 1;  // JSON-encoded types.MsgPayload
    string error = 2;
}

message FormRequest {
    string context = 1;  // JSON-encoded types.Context
    map<string, string> values = 2;
}

message FormResponse {
    string payload = 1;
    string error = 2;
}

message RulesResponse {
    string rules = 1;  // JSON-encoded []any
    string error = 2;
}

message HelpResponse {
    string help = 1;   // JSON-encoded map[string][]string
    string error = 2;
}

message IsReadyResponse {
    bool ready = 1;
}

message CallRequest {
    string operation = 1;
    string params = 2; // JSON-encoded params
}

message CallResponse {
    string result = 1; // JSON-encoded result
    string error = 2;
}

message WebhookConvertRequest {
    bytes payload = 1;
}

message OAuthCallbackRequest {
    string state = 1;
    string code = 2;
}

// Host API messages

message GetConfigRequest {
    string key = 1;
}

message GetConfigResponse {
    string value = 1;
    string error = 2;
}

message LogRequest {
    string level = 1;
    string message = 2;
    map<string, string> fields = 3;
}

message KVGetRequest {
    string key = 1;
}

message KVGetResponse {
    bytes value = 1;
    string error = 2;
}

message KVSetRequest {
    string key = 1;
    bytes value = 2;
}

message KVDeleteRequest {
    string key = 1;
}

message HTTPCallRequest {
    string method = 1;
    string url = 2;
    map<string, string> headers = 3;
    bytes body = 4;
}

message HTTPCallResponse {
    int32 status = 1;
    map<string, string> headers = 2;
    bytes body = 3;
    string error = 4;
}

message EmitEventRequest {
    string source = 1;
    string event_type = 2;
    string payload = 3; // JSON-encoded
}
```

- [ ] **Step 3: Generate protobuf Go code**

Run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/plugin/grpc/proto/plugin.proto`
Expected: generates `plugin.pb.go` and `plugin_grpc.pb.go` in the proto directory.

- [ ] **Step 4: Commit**

```bash
git add pkg/plugin/grpc/proto/plugin.proto pkg/plugin/grpc/proto/plugin.pb.go pkg/plugin/grpc/proto/plugin_grpc.pb.go
git commit -m "feat(plugin): add gRPC protobuf service definitions for PluginService and HostService"
```

---

### Task 5: gRPC Runner implementation

**Files:**
- Create: `pkg/plugin/grpc/runner.go`
- Create: `pkg/plugin/grpc/host.go`
- Create: `pkg/plugin/grpc/plugin.go`

- [ ] **Step 1: Implement GrpcRunner in `pkg/plugin/grpc/runner.go`**

```go
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/plugin/grpc/proto/pb"
)

const (
	handshakeVersion  = 1
	magicCookieKey    = "FLOWBOT_PLUGIN"
	magicCookieValue  = "flowbot-plugin-v1"
	drainTimeout      = 30 * time.Second
)

// GrpcRunner implements plugin.Runner using hashicorp/go-plugin.
type GrpcRunner struct {
	client   *plugin.Client
	svc      PluginClient
	hostSvc  *HostServer
	manifest *plugin.Manifest
	info     *plugin.PluginInfo
	inflight sync.WaitGroup
	mu       sync.Mutex
	started  bool
}

// PluginClient is the interface the go-plugin client exposes.
type PluginClient interface {
	pb.PluginServiceClient
	pb.HostServiceClient
}

// NewGrpcRunner creates a gRPC runner for a plugin manifest.
func NewGrpcRunner(m *plugin.Manifest) (*GrpcRunner, error) {
	cmd := exec.Command(m.GRPC.Binary, m.GRPC.Args...)
	// Linux: parent-death signal to prevent orphans
	setPdeathsig(cmd)

	hostSrv := &HostServer{}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  handshakeVersion,
			MagicCookieKey:   magicCookieKey,
			MagicCookieValue: magicCookieValue,
		},
		Plugins: map[string]plugin.Plugin{
			"module": &GrpcPlugin{impl: hostSrv},
		},
		Cmd:              cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Stderr:           os.Stderr,
		Stdout:           os.Stdout,
		Logger:           &hclogAdapter{},
	})

	return &GrpcRunner{
		client:  client,
		hostSvc: hostSrv,
		manifest: m,
	}, nil
}

// Load connects to the plugin and retrieves its PluginInfo.
func (r *GrpcRunner) Load(ctx context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	rpcClient, err := r.client.Client()
	if err != nil {
		return nil, fmt.Errorf("grpc load: connect failed: %w", err)
	}

	raw, err := rpcClient.Dispense("module")
	if err != nil {
		rpcClient.Close()
		return nil, fmt.Errorf("grpc load: dispense failed: %w", err)
	}

	conn, err := rpcClient.(*plugin.GRPCClient).Conn()
	if err != nil {
		return nil, fmt.Errorf("grpc load: get conn: %w", err)
	}
	r.svc = NewPluginServiceClient(conn)

	r.info = &plugin.PluginInfo{
		Name:         m.Name,
		Version:      m.Version,
		Provides:     m.Provides,
		ConfigSchema: m.ConfigSchema,
	}
	return r.info, nil
}

// Start initializes the plugin via the Init() RPC.
func (r *GrpcRunner) Start(ctx context.Context, config json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.svc.Init(ctx, &pb.InitRequest{Config: string(config)})
	if err != nil {
		return fmt.Errorf("grpc start: init failed: %w", err)
	}
	r.started = true
	return nil
}

// Stop gracefully drains in-flight calls and kills the plugin process.
func (r *GrpcRunner) Stop(ctx context.Context) error {
	r.mu.Lock()
	r.started = false
	r.mu.Unlock()

	// Drain in-flight calls
	done := make(chan struct{})
	go func() {
		r.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		// Timeout reached
	}

	r.client.Kill()
	return nil
}

// Call invokes a named function on the plugin.
func (r *GrpcRunner) Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error) {
	r.inflight.Add(1)
	defer r.inflight.Done()

	r.mu.Lock()
	started := r.started
	r.mu.Unlock()
	if !started {
		return nil, fmt.Errorf("grpc call: plugin not started")
	}

	switch function {
	case "command":
		req := &pb.CommandRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.Command(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc command: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc command: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Payload), nil

	case "form":
		req := &pb.FormRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.Form(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc form: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc form: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Payload), nil

	case "rules":
		resp, err := r.svc.Rules(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("grpc rules: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc rules: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Rules), nil

	case "help":
		resp, err := r.svc.Help(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("grpc help: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc help: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Help), nil

	case "bootstrap":
		_, err := r.svc.Bootstrap(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("grpc bootstrap: %w", err)
		}
		return nil, nil

	case "ability_call":
		req := &pb.CallRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.AbilityCall(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc ability_call: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc ability_call: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Result), nil

	case "webhook_convert":
		var raw struct {
			Payload []byte `json:"payload"`
		}
		if err := sonic.Unmarshal(params, &raw); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.WebhookConvert(ctx, &pb.WebhookConvertRequest{Payload: raw.Payload})
		if err != nil {
			return nil, fmt.Errorf("grpc webhook_convert: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc webhook_convert: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Result), nil

	default:
		return nil, fmt.Errorf("grpc call: unknown function %q", function)
	}
}

// Health checks the plugin's readiness.
func (r *GrpcRunner) Health(ctx context.Context) (*plugin.HealthStatus, error) {
	resp, err := r.svc.IsReady(ctx, &emptypb.Empty{})
	if err != nil {
		return &plugin.HealthStatus{Ready: false, LastError: err.Error()}, nil
	}
	return &plugin.HealthStatus{Ready: resp.Ready}, nil
}
```

Imports needed: `"encoding/json"`, `"fmt"`, `"os"`, `"os/exec"`, `"sync"`, `"time"`, `"github.com/bytedance/sonic"`, `"github.com/hashicorp/go-plugin"`, `"google.golang.org/grpc"`, `"google.golang.org/protobuf/types/known/emptypb"`, `"flowbot.dev/pkg/plugin"`, `"flowbot.dev/pkg/plugin/grpc/proto/pb"`.

- [ ] **Step 2: Create platform-specific pdeathsig file `pkg/plugin/grpc/runner_unix.go`**

```go
//go:build linux

package grpc

import (
	"os/exec"
	"syscall"
)

func setPdeathsig(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
```

- [ ] **Step 3: Create platform fallback `pkg/plugin/grpc/runner_other.go`**

```go
//go:build !linux

package grpc

import "os/exec"

func setPdeathsig(cmd *exec.Cmd) {}
```

- [ ] **Step 4: Implement HostServer in `pkg/plugin/grpc/host.go`**

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"flowbot.dev/pkg/plugin/grpc/proto/pb"
)

// HostServer implements pb.HostServiceServer and delegates to a plugin.HostAPI.
type HostServer struct {
	pb.UnimplementedHostServiceServer
	api plugin.HostAPI
}

// Register registers this server on a gRPC server and sets the HostAPI.
func (h *HostServer) Register(s *grpc.Server, api plugin.HostAPI) {
	h.api = api
	pb.RegisterHostServiceServer(s, h)
}

func (h *HostServer) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	val, err := h.api.GetConfig(ctx, req.Key)
	if err != nil {
		return &pb.GetConfigResponse{Error: err.Error()}, nil
	}
	return &pb.GetConfigResponse{Value: val}, nil
}

func (h *HostServer) Log(ctx context.Context, req *pb.LogRequest) (*emptypb.Empty, error) {
	h.api.Log(ctx, req.Level, req.Message, req.Fields)
	return &emptypb.Empty{}, nil
}

func (h *HostServer) KVGet(ctx context.Context, req *pb.KVGetRequest) (*pb.KVGetResponse, error) {
	val, err := h.api.KVGet(ctx, req.Key)
	if err != nil {
		return &pb.KVGetResponse{Error: err.Error()}, nil
	}
	return &pb.KVGetResponse{Value: val}, nil
}

func (h *HostServer) KVSet(ctx context.Context, req *pb.KVSetRequest) (*emptypb.Empty, error) {
	err := h.api.KVSet(ctx, req.Key, req.Value)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	return &emptypb.Empty{}, nil
}

func (h *HostServer) KVDelete(ctx context.Context, req *pb.KVDeleteRequest) (*emptypb.Empty, error) {
	err := h.api.KVDelete(ctx, req.Key)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	return &emptypb.Empty{}, nil
}

func (h *HostServer) HTTPRequest(ctx context.Context, req *pb.HTTPCallRequest) (*pb.HTTPCallResponse, error) {
	resp, err := h.api.HTTPRequest(ctx, &plugin.HostHTTPRequest{
		Method:  req.Method,
		URL:     req.Url,
		Headers: req.Headers,
		Body:    req.Body,
	})
	if err != nil {
		return &pb.HTTPCallResponse{Error: err.Error()}, nil
	}
	return &pb.HTTPCallResponse{
		Status:  int32(resp.Status),
		Headers: resp.Headers,
		Body:    resp.Body,
	}, nil
}

func (h *HostServer) EmitEvent(ctx context.Context, req *pb.EmitEventRequest) (*emptypb.Empty, error) {
	event := types.DataEvent{
		Source:    req.Source,
		EventType: req.EventType,
		Payload:   json.RawMessage(req.Payload),
	}
	err := h.api.EmitEvent(ctx, event)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	return &emptypb.Empty{}, nil
}
```

- [ ] **Step 5: Implement GrpcPlugin in `pkg/plugin/grpc/plugin.go`**

```go
package grpc

import (
	"context"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// GrpcPlugin is the hashicorp/go-plugin adapter.
type GrpcPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	impl *HostServer
}

func (p *GrpcPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	p.impl.Register(s, nil) // API set later by PluginManager
	return nil
}

func (p *GrpcPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return NewPluginServiceClient(c), nil
}

// hclogAdapter adapts go-plugin's hclog to zerolog.
type hclogAdapter struct{}

func (h *hclogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	// Use flowbot's logger
}

func (h *hclogAdapter) Trace(msg string, args ...interface{})  {}
func (h *hclogAdapter) Debug(msg string, args ...interface{})  {}
func (h *hclogAdapter) Info(msg string, args ...interface{})   {}
func (h *hclogAdapter) Warn(msg string, args ...interface{})   {}
func (h *hclogAdapter) Error(msg string, args ...interface{})  {}
func (h *hclogAdapter) IsTrace() bool { return false }
func (h *hclogAdapter) IsDebug() bool { return false }
func (h *hclogAdapter) IsInfo() bool  { return true }
func (h *hclogAdapter) IsWarn() bool  { return true }
func (h *hclogAdapter) IsError() bool { return true }
func (h *hclogAdapter) SetLevel(level hclog.Level) {}
func (h *hclogAdapter) With(args ...interface{}) hclog.Logger { return h }
func (h *hclogAdapter) Named(name string) hclog.Logger { return h }
func (h *hclogAdapter) ResetNamed(name string) hclog.Logger { return h }
func (h *hclogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger { return log.Default() }
func (h *hclogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer { return os.Stderr }
```

- [ ] **Step 6: Commit**

```bash
git add pkg/plugin/grpc/
git commit -m "feat(plugin): implement gRPC runner with hashicorp/go-plugin (host, plugin server, pdeathsig)"
```

---

### Task 6: gRPC Runner tests

**Files:**
- Create: `pkg/plugin/grpc/runner_test.go`

- [ ] **Step 1: Write unit tests for GrpcRunner in `pkg/plugin/grpc/runner_test.go`**

```go
package grpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flowbot.dev/pkg/plugin"
)

func TestNewGrpcRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		manifest *plugin.Manifest
		wantErr string
	}{
		{
			name: "valid grpc manifest creates runner",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeGRPC,
				GRPC:    &plugin.GRPCConfig{Binary: "/nonexistent/plugin", Args: []string{}},
			},
		},
		{
			name: "missing grpc config",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeGRPC,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.manifest.GRPC == nil {
				return // NewGrpcRunner expects manifest validated first
			}
			runner, err := NewGrpcRunner(tt.manifest)
			require.NoError(t, err)
			assert.NotNil(t, runner)
			assert.NotNil(t, runner.client)
		})
	}
}

func TestGrpcRunnerHealthUnconnected(t *testing.T) {
	t.Parallel()

	runner := &GrpcRunner{started: false}
	// Health of unconnected runner should fail
	_, err := runner.Call(context.Background(), "command", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}
```

- [ ] **Step 2: Run gRPC runner tests**

Run: `go test ./pkg/plugin/grpc/ -v -count=1`
Expected: Tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/plugin/grpc/runner_test.go
git commit -m "test(plugin): add GrpcRunner unit tests"
```

---

## Phase 3: Wasm Runner (wazero/wazero)

### Task 7: Wasm memory helpers

**Files:**
- Create: `pkg/plugin/wasm/memory.go`
- Create: `pkg/plugin/wasm/memory_test.go`

- [ ] **Step 1: Add wazero dependency**

Run: `go get github.com/wazero/wazero`
Expected: dependency added to go.mod as direct.

- [ ] **Step 2: Implement memory read/write helpers in `pkg/plugin/wasm/memory.go`**

```go
package wasm

import (
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic"
	wazeroapi "github.com/wazero/wazero/experimental/gojs/api"
)

// writeJSON writes JSON data to the wasm module's memory.
// Returns pointer and size for the allocated buffer.
func writeJSON(mod wazeroapi.Module, data any) (uint32, uint32, error) {
	raw, err := sonic.Marshal(data)
	if err != nil {
		return 0, 0, fmt.Errorf("marshal: %w", err)
	}
	return writeBytes(mod, raw)
}

// writeBytes writes raw bytes to the wasm module's memory.
func writeBytes(mod wazeroapi.Module, data []byte) (uint32, uint32, error) {
	size := uint32(len(data))
	if size == 0 {
		return 0, 0, nil
	}
	allocFn := mod.ExportedFunction("alloc")
	results, err := allocFn.Call(mod, uint64(size))
	if err != nil {
		return 0, 0, fmt.Errorf("alloc: %w", err)
	}
	ptr := uint32(results[0])
	if ptr == 0 {
		return 0, 0, fmt.Errorf("alloc returned null pointer")
	}
	if !mod.Memory().Write(ptr, data) {
		return 0, 0, fmt.Errorf("memory write failed at ptr=%d size=%d", ptr, size)
	}
	return ptr, size, nil
}

// readJSON reads a JSON response from wasm memory.
// result is the raw i64 return value encoding (ptr << 32) | size.
func readJSON(mod wazeroapi.Module, result uint64, target any) error {
	ptr, size := decodeResult(result)
	if size == 0 {
		return nil
	}
	data, ok := mod.Memory().Read(ptr, size)
	if !ok {
		return fmt.Errorf("memory read failed at ptr=%d size=%d", ptr, size)
	}

	// Free the buffer in wasm memory
	freeFn := mod.ExportedFunction("free")
	if freeFn != nil {
		go func() {
			freeFn.Call(mod, uint64(ptr))
		}()
	}

	// Decode JSON envelope: {"error": "...", "data": ...}
	var envelope struct {
		Error *string         `json:"error"`
		Data  json.RawMessage `json:"data"`
	}
	if err := sonic.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("plugin error: %s", *envelope.Error)
	}
	if err := sonic.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}
	return nil
}

// decodeResult decodes (ptr << 32) | size
func decodeResult(result uint64) (uint32, uint32) {
	ptr := uint32(result >> 32)
	size := uint32(result & 0xFFFFFFFF)
	return ptr, size
}
```

- [ ] **Step 3: Write memory tests in `pkg/plugin/wasm/memory_test.go`**

```go
package wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   uint64
		wantPtr  uint32
		wantSize uint32
	}{
		{
			name:     "zero result",
			result:   0,
			wantPtr:  0,
			wantSize: 0,
		},
		{
			name:     "normal result",
			result:   (1024 << 32) | 256,
			wantPtr:  1024,
			wantSize: 256,
		},
		{
			name:     "max uint32 values",
			result:   (4294967295 << 32) | 4294967295,
			wantPtr:  4294967295,
			wantSize: 4294967295,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr, size := decodeResult(tt.result)
			assert.Equal(t, tt.wantPtr, ptr)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}
```

- [ ] **Step 4: Run memory tests**

Run: `go test ./pkg/plugin/wasm/ -run TestDecodeResult -v -count=1`
Expected: Tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/plugin/wasm/
git commit -m "feat(plugin): add Wasm memory helpers (readJSON, writeJSON, decodeResult)"
```

---

### Task 8: Wasm Runner implementation

**Files:**
- Create: `pkg/plugin/wasm/runner.go`
- Create: `pkg/plugin/wasm/host.go`

- [ ] **Step 1: Implement WasmRunner in `pkg/plugin/wasm/runner.go`**

```go
package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wazero/wazero"
	"github.com/wazero/wazero/imports/wasi_snapshot_preview1"
	wazeroapi "github.com/wazero/wazero/experimental/gojs/api"

	"flowbot.dev/pkg/plugin"
)

// WasmRunner implements plugin.Runner using wazero/wazero.
type WasmRunner struct {
	manifest   *plugin.Manifest
	runtime    wazero.Runtime
	compiled   wazero.CompiledModule
	instance   atomic.Pointer[wazeroapi.Module]
	hostBindings *HostBindings
	timeout    time.Duration
	memMax     uint32 // in bytes

	// Instance pool
	pool       sync.Pool
	poolInflight sync.WaitGroup
	poolSize   int
	poolTimeout time.Duration

	inflight sync.WaitGroup
	started  atomic.Bool
}

// NewWasmRunner creates a Wasm runner for a plugin manifest.
func NewWasmRunner(m *plugin.Manifest) (*WasmRunner, error) {
	ctx := context.Background()
	wasmCfg := m.Wasm

	timeout := 30 * time.Second
	if wasmCfg.Permissions != nil && wasmCfg.Permissions.Execution != nil {
		if d, err := time.ParseDuration(wasmCfg.Permissions.Execution.Timeout); err == nil {
			timeout = d
		}
	}

	memMax := uint32(64 * 1024 * 1024) // 64MB default
	if wasmCfg.Permissions != nil && wasmCfg.Permissions.Memory != nil {
		// Parse memory limit string like "64MB"
		// For now, use the default
	}

	poolSize := 4
	poolTimeout := 5 * time.Second
	if wasmCfg.Pool != nil {
		if wasmCfg.Pool.MaxInstances > 0 {
			poolSize = wasmCfg.Pool.MaxInstances
		}
		if wasmCfg.Pool.WaitTimeout != "" {
			if d, err := time.ParseDuration(wasmCfg.Pool.WaitTimeout); err == nil {
				poolTimeout = d
			}
		}
	}

	bindings := &HostBindings{
		httpPerms: buildAllowlist(wasmCfg.Permissions),
	}

	r := wazero.NewRuntime(ctx)
	hostAPI := bindings.exportToRuntime(r)

	return &WasmRunner{
		manifest:    m,
		runtime:     r,
		hostBindings: bindings,
		timeout:     timeout,
		memMax:      memMax,
		poolSize:    poolSize,
		poolTimeout: poolTimeout,
	}, nil
}

// Load compiles the wasm module and pre-instantiates the instance pool.
func (r *WasmRunner) Load(ctx context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	wasmBytes, err := os.ReadFile(m.Wasm.Module)
	if err != nil {
		return nil, fmt.Errorf("wasm load: read module: %w", err)
	}

	compiled, err := r.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("wasm load: compile: %w", err)
	}
	r.compiled = compiled

	// Pre-instantiate pool
	r.pool.New = func() any {
		mod, err := r.runtime.InstantiateModule(ctx, compiled, r.moduleConfig())
		if err != nil {
			return nil
		}
		return mod
	}

	// Create primary instance
	mod, err := r.runtime.InstantiateModule(ctx, compiled, r.moduleConfig())
	if err != nil {
		return nil, fmt.Errorf("wasm load: instantiate: %w", err)
	}
	r.instance.Store(&mod)

	info := &plugin.PluginInfo{
		Name:         m.Name,
		Version:      m.Version,
		Provides:     m.Provides,
		ConfigSchema: m.ConfigSchema,
	}
	return info, nil
}

// moduleConfig returns the wazero ModuleConfig with sandboxed permissions.
func (r *WasmRunner) moduleConfig() wazero.ModuleConfig {
	cfg := wazero.NewModuleConfig().
		WithName("").
		WithSysNanotime().
		WithSysWalltime().
		WithRandSource(cryptoRandSource{})

	if r.memMax > 0 {
		cfg = cfg.WithMemoryLimit(r.memMax)
	}

	// Mount filesystem from permissions
	if r.manifest.Wasm.Permissions != nil {
		for _, fs := range r.manifest.Wasm.Permissions.Filesystem {
			cfg = cfg.WithFSConfig(wazero.NewFSConfig().WithDirMount(fs.Path, "/"))
		}
	}

	return cfg
}

// Start calls the wasm init() export.
func (r *WasmRunner) Start(ctx context.Context, config json.RawMessage) error {
	mod := *r.instance.Load()
	ptr, size, err := writeJSON(mod, config)
	if err != nil {
		return fmt.Errorf("wasm start: %w", err)
	}

	result, err := mod.ExportedFunction("init").Call(ctx, uint64(ptr), uint64(size))
	if err != nil {
		return fmt.Errorf("wasm start: init call: %w", err)
	}

	var resp any
	if err := readJSON(mod, result[0], &resp); err != nil {
		return fmt.Errorf("wasm start: result: %w", err)
	}

	r.started.Store(true)
	return nil
}

// Stop drains in-flight calls and closes all wasm instances.
func (r *WasmRunner) Stop(ctx context.Context) error {
	r.started.Store(false)

	done := make(chan struct{})
	go func() {
		r.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}

	if mod := r.instance.Load(); mod != nil {
		(*mod).Close(ctx)
	}
	r.runtime.Close(ctx)
	return nil
}

// Call invokes a named wasm export.
func (r *WasmRunner) Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error) {
	if !r.started.Load() {
		return nil, fmt.Errorf("wasm call: plugin not started")
	}

	r.inflight.Add(1)
	defer r.inflight.Done()

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Acquire an instance from pool or borrow the primary
	mod := r.acquireInstance()
	if mod == nil {
		return nil, fmt.Errorf("wasm call: no available instance")
	}
	defer r.releaseInstance(mod)

	// Map function name to wasm export
	exportName := function
	switch function {
	case "ability_call":
		exportName = "ability_call"
	case "webhook_convert":
		exportName = "webhook_convert"
	case "oauth_authorize":
		exportName = "oauth_authorize"
	case "oauth_callback":
		exportName = "oauth_callback"
	}

	fn := mod.ExportedFunction(exportName)
	if fn == nil {
		return nil, fmt.Errorf("wasm call: export %q not found", exportName)
	}

	// Write params to memory and call
	ptr, size, err := writeBytes(mod, params)
	if err != nil {
		return nil, fmt.Errorf("wasm call: write params: %w", err)
	}

	args := []uint64{uint64(ptr), uint64(size)}
	results, err := fn.Call(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("wasm call %q: %w", function, err)
	}

	var raw json.RawMessage
	if err := readJSON(mod, results[0], &raw); err != nil {
		return nil, fmt.Errorf("wasm call %q: read result: %w", function, err)
	}
	return raw, nil
}

// Health checks the plugin's is_ready export.
func (r *WasmRunner) Health(ctx context.Context) (*plugin.HealthStatus, error) {
	mod := r.acquireInstance()
	if mod == nil {
		return &plugin.HealthStatus{Ready: false, LastError: "no instance available"}, nil
	}
	defer r.releaseInstance(mod)

	fn := mod.ExportedFunction("is_ready")
	if fn == nil {
		return &plugin.HealthStatus{Ready: false, LastError: "is_ready export not found"}, nil
	}

	results, err := fn.Call(ctx)
	if err != nil {
		return &plugin.HealthStatus{Ready: false, LastError: err.Error()}, nil
	}

	return &plugin.HealthStatus{Ready: results[0] == 1}, nil
}

// acquireInstance borrows an instance from the pool or primary.
func (r *WasmRunner) acquireInstance() wazeroapi.Module {
	// Try from pool first
	obj := r.pool.Get()
	if obj != nil {
		if mod, ok := obj.(wazeroapi.Module); ok && mod != nil {
			return mod
		}
	}
	// Fallback to primary
	if inst := r.instance.Load(); inst != nil {
		return *inst
	}
	return nil
}

// releaseInstance returns an instance to the pool or no-ops for primary.
func (r *WasmRunner) releaseInstance(mod wazeroapi.Module) {
	r.pool.Put(mod)
}
```

- [ ] **Step 2: Implement host bindings in `pkg/plugin/wasm/host.go`**

```go
package wasm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/wazero/wazero"
	wazeroapi "github.com/wazero/wazero/experimental/gojs/api"

	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/types"
)

// HostBindings implements the host function imports for wasm plugins.
type HostBindings struct {
	mu        sync.RWMutex
	api       plugin.HostAPI
	httpPerms map[string]bool // allowed HTTP hosts
}

// SetAPI sets the HostAPI for the bindings.
func (h *HostBindings) SetAPI(api plugin.HostAPI) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.api = api
}

// exportToRuntime registers all host functions into the wazero runtime.
func (h *HostBindings) exportToRuntime(r wazero.Runtime) {
	r.NewHostModuleBuilder("flowbot").
		NewFunctionBuilder().WithFunc(h.getConfig).Export("get_config").
		NewFunctionBuilder().WithFunc(h.log).Export("log").
		NewFunctionBuilder().WithFunc(h.kvGet).Export("kv_get").
		NewFunctionBuilder().WithFunc(h.kvSet).Export("kv_set").
		NewFunctionBuilder().WithFunc(h.kvDelete).Export("kv_delete").
		NewFunctionBuilder().WithFunc(h.httpRequest).Export("http_request").
		NewFunctionBuilder().WithFunc(h.emitEvent).Export("emit_event")
}

func (h *HostBindings) getConfig(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize, outPtr uint32) uint32 {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return 0
	}
	key := readString(mod, keyPtr, keySize)
	val, err := api.GetConfig(ctx, key)
	if err != nil {
		return 0
	}
	mod.Memory().Write(outPtr, []byte(val))
	return uint32(len(val))
}

func (h *HostBindings) log(ctx context.Context, mod wazeroapi.Module, level, msgPtr, msgSize uint32) {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return
	}
	msg := readString(mod, msgPtr, msgSize)
	api.Log(ctx, levelToString(level), msg, nil)
}

func (h *HostBindings) kvGet(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize uint32) uint64 {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return 0
	}
	key := readString(mod, keyPtr, keySize)
	val, err := api.KVGet(ctx, key)
	if err != nil {
		return encodeResult(0, 0)
	}
	ptr, size, err := writeBytes(mod, val)
	if err != nil {
		return encodeResult(0, 0)
	}
	return encodeResult(ptr, size)
}

func (h *HostBindings) kvSet(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize, valPtr, valSize uint32) uint32 {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return 0
	}
	key := readString(mod, keyPtr, keySize)
	data, _ := mod.Memory().Read(valPtr, valSize)
	if err := api.KVSet(ctx, key, data); err != nil {
		return 0
	}
	return 1
}

func (h *HostBindings) kvDelete(ctx context.Context, mod wazeroapi.Module, keyPtr, keySize uint32) uint32 {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return 0
	}
	key := readString(mod, keyPtr, keySize)
	if err := api.KVDelete(ctx, key); err != nil {
		return 0
	}
	return 1
}

func (h *HostBindings) httpRequest(ctx context.Context, mod wazeroapi.Module, reqPtr, reqSize uint32) uint64 {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return 0
	}
	data, _ := mod.Memory().Read(reqPtr, reqSize)
	var req plugin.HostHTTPRequest
	if err := sonic.Unmarshal(data, &req); err != nil {
		return 0
	}
	// Enforce HTTP allowlist
	if h.httpPerms != nil && !h.httpPerms[extractHost(req.URL)] {
		return 0
	}
	resp, err := api.HTTPRequest(ctx, &req)
	if err != nil {
		return 0
	}
	respJSON, _ := sonic.Marshal(resp)
	ptr, size, err := writeBytes(mod, respJSON)
	if err != nil {
		return 0
	}
	return encodeResult(ptr, size)
}

func (h *HostBindings) emitEvent(ctx context.Context, mod wazeroapi.Module, eventPtr, eventSize uint32) uint32 {
	h.mu.RLock()
	api := h.api
	h.mu.RUnlock()
	if api == nil {
		return 0
	}
	data, _ := mod.Memory().Read(eventPtr, eventSize)
	var event types.DataEvent
	if err := sonic.Unmarshal(data, &event); err != nil {
		return 0
	}
	if err := api.EmitEvent(ctx, event); err != nil {
		return 0
	}
	return 1
}

// Helper functions
func readString(mod wazeroapi.Module, ptr, size uint32) string {
	if size == 0 {
		return ""
	}
	data, ok := mod.Memory().Read(ptr, size)
	if !ok {
		return ""
	}
	return string(data)
}

func levelToString(level uint32) string {
	switch level {
	case 0: return "debug"
	case 1: return "info"
	case 2: return "warn"
	case 3: return "error"
	default: return "info"
	}
}

func extractHost(urlStr string) string {
	u, err := urlParse(urlStr)
	if err != nil {
		return ""
	}
	return u.Host
}

func encodeResult(ptr, size uint32) uint64 {
	return (uint64(ptr) << 32) | uint64(size)
}

func buildAllowlist(perms *plugin.WasmPermissions) map[string]bool {
	if perms == nil {
		return nil
	}
	al := make(map[string]bool, len(perms.HTTP))
	for _, p := range perms.HTTP {
		al[p.Host] = true
	}
	return al
}
```

- [ ] **Step 3: Commit**

```bash
git add pkg/plugin/wasm/runner.go pkg/plugin/wasm/host.go
git commit -m "feat(plugin): implement Wasm runner with wazero (runner, host bindings, pool)"
```

---

### Task 9: Wasm Runner tests

**Files:**
- Create: `pkg/plugin/wasm/runner_test.go`

- [ ] **Step 1: Write Wasm runner tests in `pkg/plugin/wasm/runner_test.go`**

```go
package wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"flowbot.dev/pkg/plugin"
)

func TestNewWasmRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		manifest *plugin.Manifest
		wantErr string
	}{
		{
			name: "valid wasm manifest creates runner",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeWasm,
				Wasm: &plugin.WasmConfig{
					Module: "./testdata/empty.wasm",
				},
			},
		},
		{
			name: "default timeout when no execution limit",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeWasm,
				Wasm:   &plugin.WasmConfig{Module: "./testdata/empty.wasm"},
			},
		},
		{
			name: "custom execution timeout",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeWasm,
				Wasm: &plugin.WasmConfig{
					Module: "./testdata/empty.wasm",
					Permissions: &plugin.WasmPermissions{
						Execution: &plugin.ExecutionLimit{Timeout: "10s"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewWasmRunner(tt.manifest)
			assert.NoError(t, err)
			assert.NotNil(t, runner)

			if tt.manifest.Wasm.Permissions != nil && tt.manifest.Wasm.Permissions.Execution != nil {
				assert.Equal(t, 10*time.Second, runner.timeout)
			} else {
				assert.Equal(t, 30*time.Second, runner.timeout)
			}
		})
	}
}

func TestWasmRunnerHealthUnstarted(t *testing.T) {
	t.Parallel()

	runner := &WasmRunner{}
	_, err := runner.Call(context.Background(), "command", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}
```

- [ ] **Step 2: Run Wasm runner tests**

Run: `go test ./pkg/plugin/wasm/ -v -count=1`
Expected: Tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/plugin/wasm/runner_test.go
git commit -m "test(plugin): add WasmRunner unit tests"
```

---

## Phase 4: Adapter Layer

### Task 10: Plugin Module Adapter

**Files:**
- Create: `pkg/plugin/adapter/module.go`
- Create: `pkg/plugin/adapter/module_test.go`

- [ ] **Step 1: Implement PluginModuleAdapter in `pkg/plugin/adapter/module.go`**

```go
package adapter

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"

	"flowbot.dev/pkg/module"
	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/types"
)

// PluginModuleAdapter implements module.Handler by delegating to a Runner.
type PluginModuleAdapter struct {
	module.Base
	runner   atomic.Pointer[plugin.Runner]
	name     string
	manifest *plugin.Manifest
	ready    atomic.Bool
}

// NewModuleAdapter creates a module adapter for a plugin.
func NewModuleAdapter(m *plugin.Manifest, r plugin.Runner) *PluginModuleAdapter {
	a := &PluginModuleAdapter{name: m.Name, manifest: m}
	a.runner.Store(&r)
	return a
}

// SwapRunner atomically swaps the underlying runner without unregistering.
func (a *PluginModuleAdapter) SwapRunner(newRunner plugin.Runner) {
	a.runner.Store(&newRunner)
}

// Runner returns the current runner (for testing).
func (a *PluginModuleAdapter) Runner() plugin.Runner {
	r := a.runner.Load()
	if r == nil {
		return nil
	}
	return *r
}

func (a *PluginModuleAdapter) Init(jsonconf json.RawMessage) error {
	r := a.runner.Load()
	if r == nil {
		return fmt.Errorf("plugin %s: no runner", a.name)
	}
	if err := (*r).Start(context.Background(), jsonconf); err != nil {
		return err
	}
	a.ready.Store(true)
	return nil
}

func (a *PluginModuleAdapter) IsReady() bool {
	return a.ready.Load()
}

func (a *PluginModuleAdapter) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return a.callRunner(context.Background(), "command", map[string]any{
		"context": ctx,
		"content": content,
	}, func(data json.RawMessage) (types.MsgPayload, error) {
		var p types.MsgPayload
		if err := sonic.Unmarshal(data, &p); err != nil {
			return types.MsgPayload{}, fmt.Errorf("unmarshal payload: %w", err)
		}
		return p, nil
	})
}

// callRunner is the common serialization/delegation helper.
func (a *PluginModuleAdapter) callRunner(ctx context.Context, fn string, params any, decode func(json.RawMessage) (types.MsgPayload, error)) (types.MsgPayload, error) {
	r := a.runner.Load()
	if r == nil {
		return types.MsgPayload{}, fmt.Errorf("plugin %s: no runner", a.name)
	}

	raw, err := sonic.Marshal(params)
	if err != nil {
		return types.MsgPayload{}, fmt.Errorf("marshal params: %w", err)
	}

	result, err := (*r).Call(ctx, fn, raw)
	if err != nil {
		return types.MsgPayload{}, fmt.Errorf("plugin %s call %s: %w", a.name, fn, err)
	}

	return decode(result)
}

// decodePayload is the default payload decoder used by Form, etc.
func decodePayload(data json.RawMessage) (types.MsgPayload, error) {
	var p types.MsgPayload
	if err := sonic.Unmarshal(data, &p); err != nil {
		return types.MsgPayload{}, fmt.Errorf("unmarshal payload: %w", err)
	}
	return p, nil
}

// Form, Rules, Help, Bootstrap — same pattern using callRunner
func (a *PluginModuleAdapter) Form(ctx types.Context, values map[string]string) (types.MsgPayload, error) {
	return a.callRunner(context.Background(), "form", map[string]any{
		"context": ctx,
		"values":  values,
	}, decodePayload)
}

func (a *PluginModuleAdapter) Rules() []any {
	r := a.runner.Load()
	if r == nil {
		return nil
	}
	result, err := (*r).Call(context.Background(), "rules", nil)
	if err != nil {
		return nil
	}
	var rules []any
	sonic.Unmarshal(result, &rules)
	return rules
}

func (a *PluginModuleAdapter) Help() (map[string][]string, error) {
	r := a.runner.Load()
	if r == nil {
		return nil, fmt.Errorf("plugin %s: no runner", a.name)
	}
	result, err := (*r).Call(context.Background(), "help", nil)
	if err != nil {
		return nil, err
	}
	var help map[string][]string
	if err := sonic.Unmarshal(result, &help); err != nil {
		return nil, fmt.Errorf("unmarshal help: %w", err)
	}
	return help, nil
}

func (a *PluginModuleAdapter) Bootstrap() error {
	r := a.runner.Load()
	if r == nil {
		return fmt.Errorf("plugin %s: no runner", a.name)
	}
	_, err := (*r).Call(context.Background(), "bootstrap", nil)
	return err
}

// Webservice is handled via proxy route, not direct mounting
func (a *PluginModuleAdapter) Webservice(app *fiber.App) {
	// Plugin webservice routes are proxied through /service/{plugin-name}/*
	// in the hub_plugins handler, not mounted here
}
```

Imports needed: `"context"`, `"encoding/json"`, `"fmt"`, `"sync/atomic"`, `"github.com/bytedance/sonic"`, `"github.com/gofiber/fiber/v3"`, `"flowbot.dev/pkg/module"`, `"flowbot.dev/pkg/plugin"`, `"flowbot.dev/pkg/types"`.

- [ ] **Step 2: Write module adapter tests in `pkg/plugin/adapter/module_test.go`**

```go
package adapter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/types"
)

// stubRunner implements plugin.Runner for testing.
type stubRunner struct {
	callFn func(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error)
}

func (s *stubRunner) Load(ctx context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	return &plugin.PluginInfo{Name: m.Name, Version: m.Version}, nil
}
func (s *stubRunner) Start(ctx context.Context, config json.RawMessage) error { return nil }
func (s *stubRunner) Stop(ctx context.Context) error                          { return nil }
func (s *stubRunner) Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error) {
	if s.callFn != nil {
		return s.callFn(ctx, function, params)
	}
	return json.RawMessage(`{}`), nil
}
func (s *stubRunner) Health(ctx context.Context) (*plugin.HealthStatus, error) {
	return &plugin.HealthStatus{Ready: true}, nil
}

func TestModuleAdapterCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     string
		wantErr    string
		wantText   string
	}{
		{
			name:     "happy path command",
			result:   `{"text": "hello from plugin"}`,
			wantText: "hello from plugin",
		},
		{
			name:    "plugin returns error",
			result:  `{}`,
			wantErr: "simulated error",
		},
		{
			name:    "nil runner",
			wantErr: "no runner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r plugin.Runner
			if tt.wantErr == "no runner" {
				// Leave runner nil
			} else {
				r = &stubRunner{
					callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
						if tt.wantErr != "" {
							return nil, fmt.Errorf(tt.wantErr)
						}
						return json.RawMessage(tt.result), nil
					},
				}
			}

			m := &plugin.Manifest{Name: "test"}
			adapter := NewModuleAdapter(m, r)

			payload, err := adapter.Command(types.Context{}, "hello")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, payload.Text)
		})
	}
}

func TestModuleAdapterSwapRunner(t *testing.T) {
	t.Parallel()

	runner1 := &stubRunner{callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"text": "runner1"}`), nil
	}}
	runner2 := &stubRunner{callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"text": "runner2"}`), nil
	}}

	adapter := NewModuleAdapter(&plugin.Manifest{Name: "test"}, runner1)
	payload, _ := adapter.Command(types.Context{}, "hello")
	assert.Equal(t, "runner1", payload.Text)

	adapter.SwapRunner(runner2)
	payload, _ = adapter.Command(types.Context{}, "hello")
	assert.Equal(t, "runner2", payload.Text)
}
```

- [ ] **Step 3: Run module adapter tests**

Run: `go test ./pkg/plugin/adapter/ -run TestModuleAdapter -v -count=1`
Expected: Tests PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/plugin/adapter/module.go pkg/plugin/adapter/module_test.go
git commit -m "feat(plugin): implement PluginModuleAdapter with atomic runner swap"
```

---

### Task 11: Plugin Ability Adapter

**Files:**
- Create: `pkg/plugin/adapter/ability.go`
- Create: `pkg/plugin/adapter/ability_test.go`

- [ ] **Step 1: Implement PluginAbilityAdapter in `pkg/plugin/adapter/ability.go`**

```go
package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic"

	"flowbot.dev/pkg/ability"
	"flowbot.dev/pkg/hub"
	"flowbot.dev/pkg/plugin"
)

// PluginAbilityAdapter registers Invoker closures for declared operations.
type PluginAbilityAdapter struct {
	runner     plugin.Runner
	capability hub.CapabilityType
	operations []string
	descriptor *hub.Descriptor
}

// NewAbilityAdapter creates an ability adapter from a manifest ability declaration.
func NewAbilityAdapter(r plugin.Runner, capType string, ops []string) *PluginAbilityAdapter {
	return &PluginAbilityAdapter{
		runner:     r,
		capability: hub.CapabilityType(capType),
		operations: ops,
	}
}

// Register registers all declared operations as ability.Invoker closures.
func (a *PluginAbilityAdapter) Register() error {
	for _, op := range a.operations {
		invoker := a.makeInvoker(op)
		if err := ability.RegisterInvoker(a.capability, op, invoker); err != nil {
			return fmt.Errorf("register invoker %s/%s: %w", a.capability, op, err)
		}
	}
	return nil
}

// Unregister removes all registered invokers.
func (a *PluginAbilityAdapter) Unregister() {
	for _, op := range a.operations {
		ability.UnregisterInvoker(a.capability, op)
	}
}

// Descriptor returns the hub.Descriptor for this ability.
func (a *PluginAbilityAdapter) Descriptor() *hub.Descriptor {
	if a.descriptor == nil {
		return nil
	}
	return a.descriptor
}

func (a *PluginAbilityAdapter) makeInvoker(op string) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		raw, err := sonic.Marshal(struct {
			Operation string         `json:"operation"`
			Params    map[string]any `json:"params"`
		}{
			Operation: op,
			Params:    params,
		})
		if err != nil {
			return nil, fmt.Errorf("ability invoke marshal: %w", err)
		}

		result, err := a.runner.Call(ctx, "ability_call", raw)
		if err != nil {
			return nil, fmt.Errorf("ability invoke: %w", err)
		}

		var invokeResult ability.InvokeResult
		if err := sonic.Unmarshal(result, &invokeResult); err != nil {
			return nil, fmt.Errorf("ability invoke unmarshal: %w", err)
		}
		return &invokeResult, nil
	}
}
```

- [ ] **Step 2: Write ability adapter tests in `pkg/plugin/adapter/ability_test.go`**

```go
package adapter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbilityAdapterRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ops     []string
		callFn  func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error)
		wantErr string
	}{
		{
			name: "registers and invokes list operation",
			ops:  []string{"list"},
			callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
				assert.Equal(t, "ability_call", fn)
				return json.RawMessage(`[{"id": "1", "name": "test"}]`), nil
			},
		},
		{
			name: "registers multiple operations",
			ops:  []string{"list", "get", "create"},
			callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{}`), nil
			},
		},
		{
			name:    "no operations is valid",
			ops:     []string{},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt
			// Unregister after test
			adapter := NewAbilityAdapter(&stubRunner{callFn: tt.callFn}, "example", tt.ops)
			err := adapter.Register()
			defer adapter.Unregister()
			if tt.wantErr != "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAbilityAdapterMakeInvoker(t *testing.T) {
	t.Parallel()

	expectedResult := `{"id": "1", "name": "item"}`
	runner := &stubRunner{
		callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(expectedResult), nil
		},
	}

	adapter := NewAbilityAdapter(runner, "example", []string{"get"})
	invoker := adapter.makeInvoker("get")

	result, err := invoker(context.Background(), map[string]any{"id": "1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
```

- [ ] **Step 3: Run ability adapter tests**

Run: `go test ./pkg/plugin/adapter/ -run TestAbilityAdapter -v -count=1`
Expected: Tests PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/plugin/adapter/ability.go pkg/plugin/adapter/ability_test.go
git commit -m "feat(plugin): implement PluginAbilityAdapter with Invoker registration"
```

---

### Task 12: Plugin Provider Adapter

**Files:**
- Create: `pkg/plugin/adapter/provider.go`
- Create: `pkg/plugin/adapter/provider_test.go`

- [ ] **Step 1: Implement PluginProviderAdapter in `pkg/plugin/adapter/provider.go`**

```go
package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic"

	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/types"
)

// PluginProviderAdapter implements provider OAuth and webhook interfaces.
type PluginProviderAdapter struct {
	runner plugin.Runner
	name   string
}

// NewProviderAdapter creates a provider adapter.
func NewProviderAdapter(r plugin.Runner, name string) *PluginProviderAdapter {
	return &PluginProviderAdapter{runner: r, name: name}
}

// GetAuthorizeURL returns the OAuth authorize URL from the plugin.
func (a *PluginProviderAdapter) GetAuthorizeURL(state string) string {
	ctx := context.Background()
	raw, err := sonic.Marshal(map[string]string{"state": state})
	if err != nil {
		return ""
	}
	result, err := a.runner.Call(ctx, "oauth_authorize", raw)
	if err != nil {
		return ""
	}
	var resp struct {
		URL string `json:"url"`
	}
	if err := sonic.Unmarshal(result, &resp); err != nil {
		return ""
	}
	return resp.URL
}

// GetAccessToken exchanges an authorization code for an access token.
func (a *PluginProviderAdapter) GetAccessToken(ctx context.Context) (*provider.OAuthToken, error) {
	return nil, fmt.Errorf("not implemented")
}

// WebhookConvert converts provider webhook payloads to DataEvents.
func (a *PluginProviderAdapter) WebhookConvert(payload []byte) ([]types.DataEvent, error) {
	raw, err := sonic.Marshal(map[string]any{"payload": payload})
	if err != nil {
		return nil, fmt.Errorf("webhook convert marshal: %w", err)
	}
	result, err := a.runner.Call(context.Background(), "webhook_convert", raw)
	if err != nil {
		return nil, fmt.Errorf("webhook convert: %w", err)
	}
	var events []types.DataEvent
	if err := sonic.Unmarshal(result, &events); err != nil {
		return nil, fmt.Errorf("webhook convert unmarshal: %w", err)
	}
	return events, nil
}
```

- [ ] **Step 2: Write provider adapter tests in `pkg/plugin/adapter/provider_test.go`**

```go
package adapter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/types"
)

func TestProviderAdapterWebhookConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		payload   []byte
		result    json.RawMessage
		wantCount int
		wantErr   string
	}{
		{
			name:      "converts single event",
			payload:   []byte(`{"type": "test"}`),
			result:    json.RawMessage(`[{"source": "plugin", "event_type": "test"}]`),
			wantCount: 1,
		},
		{
			name:      "converts multiple events",
			payload:   []byte(`{"type": "batch"}`),
			result:    json.RawMessage(`[{"source": "a"}, {"source": "b"}]`),
			wantCount: 2,
		},
		{
			name:    "converts empty result",
			payload: []byte(`{}`),
			result:  json.RawMessage(`[]`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{
				callFn: func(ctx context.Context, fn string, params json.RawMessage) (json.RawMessage, error) {
					return tt.result, nil
				},
			}
			adapter := NewProviderAdapter(runner, "test-provider")

			events, err := adapter.WebhookConvert(tt.payload)
			if tt.wantErr != "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
		})
	}
}
```

- [ ] **Step 3: Run provider adapter tests**

Run: `go test ./pkg/plugin/adapter/ -run TestProviderAdapter -v -count=1`
Expected: Tests PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/plugin/adapter/provider.go pkg/plugin/adapter/provider_test.go
git commit -m "feat(plugin): implement PluginProviderAdapter for OAuth and webhook support"
```

---

## Phase 5: Plugin Manager

### Task 13: Add Unregister methods to existing registries

**Files:**
- Modify: `pkg/module/module.go`
- Modify: `pkg/ability/invoke.go`
- Modify: `pkg/hub/registry.go`
- Modify: `pkg/providers/providers.go`

- [ ] **Step 1: Add Unregister to `pkg/module/module.go`**

```go
// Unregister removes a module handler from the global registry.
// Safe to call on non-existent names.
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(handlers, name)
}
```

Add after the `Register` function.

- [ ] **Step 2: Add UnregisterInvoker to `pkg/ability/invoke.go`**

```go
// UnregisterInvoker removes an invoker for a capability+operation.
func UnregisterInvoker(capType hub.CapabilityType, op string) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	if ops, ok := DefaultRegistry.invokers[capType]; ok {
		delete(ops, op)
		if len(ops) == 0 {
			delete(DefaultRegistry.invokers, capType)
		}
	}
}
```

- [ ] **Step 3: Add Unregister to `pkg/hub/registry.go`**

```go
// Unregister removes a capability descriptor from the hub registry.
func (r *Registry) Unregister(capType hub.CapabilityType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.descriptors, capType)
}
```

- [ ] **Step 4: Add UnregisterOAuthProvider to `pkg/providers/providers.go`**

```go
// UnregisterOAuthProvider removes an OAuth provider factory.
func UnregisterOAuthProvider(name string) {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	delete(oauthFactories, name)
}
```

- [ ] **Step 5: Run existing tests to ensure no regressions**

Run: `go test ./pkg/module/ ./pkg/ability/ ./pkg/hub/ ./pkg/providers/ -v -count=1`
Expected: All existing tests PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/module/module.go pkg/ability/invoke.go pkg/hub/registry.go pkg/providers/providers.go
git commit -m "feat(plugin): add Unregister methods to module, ability, hub, and provider registries"
```

---

### Task 14: Plugin Manager implementation

**Files:**
- Create: `pkg/plugin/manager.go`
- Create: `pkg/plugin/config.go`

- [ ] **Step 1: Implement PluginManager in `pkg/plugin/manager.go`**

```go
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"flowbot.dev/pkg/hub"
	"flowbot.dev/pkg/module"
	"flowbot.dev/pkg/plugin/adapter"
	"flowbot.dev/pkg/plugin/grpc"
	"flowbot.dev/pkg/plugin/source"
	"flowbot.dev/pkg/plugin/wasm"
	"flowbot.dev/pkg/providers"
)

// PluginManager orchestrates plugin discovery, loading, lifecycle, and hot-reload.
type PluginManager struct {
	instances map[string]*PluginInstance
	sources   []source.Source
	config    *PluginConfig
	mu        sync.RWMutex
	logger    zerolog.Logger
}

// PluginInstance tracks a loaded plugin's state.
type PluginInstance struct {
	Identity  string            // source-derived identity
	Manifest  *Manifest
	Runner    Runner
	Adapters  *PluginAdapters   // module, ability, provider adapters
	State     PluginState
	StartedAt time.Time
	LastError error
}

// PluginAdapters holds all adapters for a plugin instance.
type PluginAdapters struct {
	Module   *adapter.PluginModuleAdapter
	Abilities []*adapter.PluginAbilityAdapter
	Provider  *adapter.PluginProviderAdapter
}

// NewPluginManager creates a PluginManager from configuration.
func NewPluginManager(cfg *PluginConfig, log zerolog.Logger) *PluginManager {
	return &PluginManager{
		instances: make(map[string]*PluginInstance),
		config:    cfg,
		logger:    log.With().Str("component", "plugin-manager").Logger(),
	}
}

// Init discovers and loads all plugins from configured sources.
func (m *PluginManager) Init(ctx context.Context, pluginConfigs map[string]json.RawMessage) error {
	if m.config == nil || !m.config.Enabled {
		m.logger.Info().Msg("plugin system disabled")
		return nil
	}

	for _, srcCfg := range m.config.Sources {
		src, err := source.NewSource(srcCfg)
		if err != nil {
			m.logger.Error().Err(err).Str("source", srcCfg.Type).Msg("failed to create source")
			continue
		}
		m.sources = append(m.sources, src)
	}

	for _, src := range m.sources {
		manifests, err := src.Discover(ctx)
		if err != nil {
			m.logger.Error().Err(err).Msg("discovery failed")
			continue
		}
		for _, manifest := range manifests {
			identity := deriveIdentity(srcCfg, manifest)
			if _, exists := m.instances[identity]; exists {
				m.logger.Warn().Str("identity", identity).Msg("duplicate plugin identity, skipping")
				continue
			}

			cfg := pluginConfigs[identity]
			if err := m.loadPlugin(ctx, identity, manifest, cfg); err != nil {
				m.logger.Error().Err(err).Str("identity", identity).Msg("failed to load plugin")
			}
		}
	}
	return nil
}

// deriveIdentity creates a plugin identity from the source configuration.
// Uses directory name for local, org/repo for OCI and Git.
func deriveIdentity(srcCfg source.SourceConfig, m *Manifest) string {
	switch srcCfg.Type {
	case "local":
		return m.Name // dir name from source
	case "oci", "git":
		return m.Name // org/repo from source
	default:
		return m.Name
	}
}

// loadPlugin loads a single plugin, creates adapters, and registers them.
func (m *PluginManager) loadPlugin(ctx context.Context, identity string, manifest *Manifest, cfg json.RawMessage) error {
	m.logger.Info().Str("identity", identity).Str("runtime", string(manifest.Runtime)).Msg("loading plugin")

	// Validate config against schema
	if err := manifest.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Create runner
	var runner Runner
	var err error
	switch manifest.Runtime {
	case RuntimeGRPC:
		runner, err = grpc.NewGrpcRunner(manifest)
	case RuntimeWasm:
		runner, err = wasm.NewWasmRunner(manifest)
	default:
		return fmt.Errorf("unknown runtime: %s", manifest.Runtime)
	}
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}

	// Load
	if _, err := runner.Load(ctx, manifest); err != nil {
		return fmt.Errorf("runner load: %w", err)
	}

	// Create adapters
	adapters := &PluginAdapters{}
	if manifest.Provides.Module {
		adapters.Module = adapter.NewModuleAdapter(manifest, runner)
		module.Register(identity, adapters.Module)
	}
	for _, ab := range manifest.Provides.Abilities {
		a := adapter.NewAbilityAdapter(runner, ab.Capability, ab.Operations)
		if err := a.Register(); err != nil {
			return fmt.Errorf("register ability %s: %w", ab.Capability, err)
		}
		adapters.Abilities = append(adapters.Abilities, a)
	}
	if manifest.Provides.Provider != nil {
		adapters.Provider = adapter.NewProviderAdapter(runner, manifest.Provides.Provider.Name)
		if manifest.Provides.Provider.OAuth {
			providers.RegisterOAuthProvider(manifest.Provides.Provider.Name, func() providers.OAuthProvider {
				return adapters.Provider
			})
		}
	}

	// Start
	if err := runner.Start(ctx, cfg); err != nil {
		return fmt.Errorf("runner start: %w", err)
	}

	m.mu.Lock()
	m.instances[identity] = &PluginInstance{
		Identity:  identity,
		Manifest:  manifest,
		Runner:    runner,
		Adapters:  adapters,
		State:     StateRunning,
		StartedAt: time.Now(),
	}
	m.mu.Unlock()

	m.logger.Info().Str("identity", identity).Msg("plugin loaded successfully")
	return nil
}

// ReloadPlugin hot-reloads a plugin with validate-then-swap semantics.
func (m *PluginManager) ReloadPlugin(ctx context.Context, identity string, newManifest *Manifest, newCfg json.RawMessage) error {
	m.mu.RLock()
	old := m.instances[identity]
	m.mu.RUnlock()
	if old == nil {
		return fmt.Errorf("plugin %s not found", identity)
	}

	// Step 2: Validate new config against new schema (fail → abort)
	if err := newManifest.ValidateConfig(newCfg); err != nil {
		return fmt.Errorf("hot-reload aborted: config validation failed: %w", err)
	}

	// Create new runner
	var newRunner Runner
	var err error
	switch newManifest.Runtime {
	case RuntimeGRPC:
		newRunner, err = grpc.NewGrpcRunner(newManifest)
	case RuntimeWasm:
		newRunner, err = wasm.NewWasmRunner(newManifest)
	}
	if err != nil {
		return fmt.Errorf("create new runner: %w", err)
	}

	if _, err := newRunner.Load(ctx, newManifest); err != nil {
		return fmt.Errorf("load new runner: %w", err)
	}
	if err := newRunner.Start(ctx, newCfg); err != nil {
		return fmt.Errorf("start new runner: %w", err)
	}

	// Determine if provides changed
	providesChanged := !providesEqual(old.Manifest.Provides, newManifest.Provides)

	if providesChanged {
		// Unregister old, register new
		m.unregisterAdapters(old)
		m.registerAdapters(identity, newManifest, newRunner)
	} else {
		// Atomic swap: just swap the runner in the adapter
		if old.Adapters.Module != nil {
			old.Adapters.Module.SwapRunner(newRunner)
		}
	}

	// Drain and stop old runner
	drainCtx, cancel := context.WithTimeout(ctx, m.config.DrainTimeout)
	defer cancel()
	if err := old.Runner.Stop(drainCtx); err != nil {
		m.logger.Warn().Err(err).Str("identity", identity).Msg("old runner stop had errors")
	}

	// Swap instance
	m.mu.Lock()
	m.instances[identity] = &PluginInstance{
		Identity:  identity,
		Manifest:  newManifest,
		Runner:    newRunner,
		Adapters:  old.Adapters,
		State:     StateRunning,
		StartedAt: time.Now(),
	}
	m.mu.Unlock()

	return nil
}

// UnloadPlugin unloads a plugin and removes it from all registries.
func (m *PluginManager) UnloadPlugin(ctx context.Context, identity string) error {
	m.mu.Lock()
	inst, ok := m.instances[identity]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("plugin %s not found", identity)
	}
	delete(m.instances, identity)
	m.mu.Unlock()

	m.unregisterAdapters(inst)

	drainCtx, cancel := context.WithTimeout(ctx, m.config.DrainTimeout)
	defer cancel()
	return inst.Runner.Stop(drainCtx)
}

// List returns all loaded plugin instances.
func (m *PluginManager) List() []*PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*PluginInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		result = append(result, inst)
	}
	return result
}

// unregisterAdapters removes all adapters for a plugin instance from registries.
func (m *PluginManager) unregisterAdapters(inst *PluginInstance) {
	if inst.Adapters.Module != nil {
		module.Unregister(inst.Identity)
	}
	for _, a := range inst.Adapters.Abilities {
		a.Unregister()
	}
	if inst.Adapters.Provider != nil && inst.Manifest.Provides.Provider.OAuth {
		providers.UnregisterOAuthProvider(inst.Manifest.Provides.Provider.Name)
	}
}

// registerAdapters registers all adapters for a plugin into flowbot registries.
func (m *PluginManager) registerAdapters(identity string, manifest *Manifest, runner Runner) {
	if manifest.Provides.Module {
		adapter := adapter.NewModuleAdapter(manifest, runner)
		module.Register(identity, adapter)
	}
	for _, ab := range manifest.Provides.Abilities {
		a := adapter.NewAbilityAdapter(runner, ab.Capability, ab.Operations)
		a.Register()
	}
	if manifest.Provides.Provider != nil && manifest.Provides.Provider.OAuth {
		adapter := adapter.NewProviderAdapter(runner, manifest.Provides.Provider.Name)
		providers.RegisterOAuthProvider(manifest.Provides.Provider.Name, func() providers.OAuthProvider {
			return adapter
		})
	}
}

// providesEqual checks if two Provides declarations are equivalent.
func providesEqual(a, b Provides) bool {
	if a.Module != b.Module {
		return false
	}
	if (a.Provider == nil) != (b.Provider == nil) {
		return false
	}
	if a.Provider != nil {
		if a.Provider.Name != b.Provider.Name || a.Provider.OAuth != b.Provider.OAuth {
			return false
		}
	}
	if len(a.Abilities) != len(b.Abilities) {
		return false
	}
	for i := range a.Abilities {
		if a.Abilities[i].Capability != b.Abilities[i].Capability {
			return false
		}
		if !stringSlicesEqual(a.Abilities[i].Operations, b.Abilities[i].Operations) {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Implement PluginConfig in `pkg/plugin/config.go`**

```go
package plugin

import (
	"encoding/json"
	"time"

	"flowbot.dev/pkg/plugin/source"
)

// PluginConfig is the top-level plugins section in flowbot.yaml.
type PluginConfig struct {
	Enabled      bool                       `json:"enabled" yaml:"enabled"`
	Sources      []source.SourceConfig      `json:"sources" yaml:"sources"`
	Config       map[string]json.RawMessage `json:"config" yaml:"config"`
	HotReload    bool                       `json:"hot_reload" yaml:"hot_reload"`
	DrainTimeout time.Duration              `json:"drain_timeout" yaml:"drain_timeout"` // default 30s
	MaxPlugins   int                        `json:"max_plugins" yaml:"max_plugins"`     // default 50
}

// DefaultPluginConfig returns the default plugin configuration.
func DefaultPluginConfig() *PluginConfig {
	return &PluginConfig{
		Enabled:      false,
		HotReload:    true,
		DrainTimeout: 30 * time.Second,
		MaxPlugins:   50,
	}
}
```

Note: `SourceConfig` and `GitRepoConfig` types are defined in `pkg/plugin/source/source.go`, not duplicated in `config.go`.

- [ ] **Step 3: Commit**

```bash
git add pkg/plugin/manager.go pkg/plugin/config.go
git commit -m "feat(plugin): implement PluginManager with load, unload, and hot-reload (validate-then-swap)"
```

---

### Task 15: Plugin Manager tests

**Files:**
- Create: `pkg/plugin/manager_test.go`

- [ ] **Step 1: Write PluginManager unit tests in `pkg/plugin/manager_test.go`**

```go
package plugin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginManagerDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *PluginConfig
		wantNil bool
	}{
		{
			name: "disabled manager does nothing",
			config: &PluginConfig{Enabled: false},
		},
		{
			name:    "nil config is no-op",
			config:  nil,
		},
		{
			name: "enabled manager processes sources",
			config: &PluginConfig{
				Enabled: true,
				Sources: []SourceConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewPluginManager(tt.config, zerolog.Nop())
			assert.NotNil(t, mgr)
			// Init with no configs
			err := mgr.Init(context.Background(), nil)
			assert.NoError(t, err)
		})
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()

	mgr := NewPluginManager(DefaultPluginConfig(), zerolog.Nop())
	list := mgr.List()
	assert.Empty(t, list)
}

func TestUnloadNotFound(t *testing.T) {
	t.Parallel()

	mgr := NewPluginManager(DefaultPluginConfig(), zerolog.Nop())
	err := mgr.UnloadPlugin(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReloadNotFound(t *testing.T) {
	t.Parallel()

	mgr := NewPluginManager(DefaultPluginConfig(), zerolog.Nop())
	err := mgr.ReloadPlugin(context.Background(), "nonexistent", &Manifest{Name: "test"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProvidesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		a     Provides
		b     Provides
		equal bool
	}{
		{
			name:  "identical empty",
			a:     Provides{},
			b:     Provides{},
			equal: true,
		},
		{
			name:  "identical module only",
			a:     Provides{Module: true},
			b:     Provides{Module: true},
			equal: true,
		},
		{
			name:  "module mismatch",
			a:     Provides{Module: true},
			b:     Provides{Module: false},
			equal: false,
		},
		{
			name: "abilities match",
			a:    Provides{Abilities: []AbilityDecl{{Capability: "bookmark", Operations: []string{"list"}}}},
			b:    Provides{Abilities: []AbilityDecl{{Capability: "bookmark", Operations: []string{"list"}}}},
			equal: true,
		},
		{
			name: "abilities operations mismatch",
			a:    Provides{Abilities: []AbilityDecl{{Capability: "bookmark", Operations: []string{"list"}}}},
			b:    Provides{Abilities: []AbilityDecl{{Capability: "bookmark", Operations: []string{"list", "get"}}}},
			equal: false,
		},
		{
			name:  "provider mismatch nil vs non-nil",
			a:     Provides{Provider: nil},
			b:     Provides{Provider: &ProviderDecl{Name: "test"}},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, providesEqual(tt.a, tt.b))
		})
	}
}
```

- [ ] **Step 2: Run PluginManager tests**

Run: `go test ./pkg/plugin/ -run "TestPluginManager|TestListEmpty|TestUnloadNotFound|TestReloadNotFound|TestProvidesEqual" -v -count=1`
Expected: Tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/plugin/manager_test.go
git commit -m "test(plugin): add PluginManager unit tests (lifecycle, providesEqual, empty states)"
```

---

## Phase 6: Distribution Sources

### Task 16: Source interface and local source

**Files:**
- Create: `pkg/plugin/source/source.go`
- Create: `pkg/plugin/source/local.go`
- Create: `pkg/plugin/source/local_test.go`

- [ ] **Step 1: Define Source interface and SourceConfig in `pkg/plugin/source/source.go`**

```go
package source

import (
	"context"

	"flowbot.dev/pkg/plugin"
)

// SourceConfig matches the YAML config for a single source entry.
type SourceConfig struct {
	Type         string   `json:"type" yaml:"type"`
	Path         string   `json:"path" yaml:"path"`
	Registry     string   `json:"registry" yaml:"registry"`
	Repos        []GitRepoConfig `json:"repos" yaml:"repos"`
	PollInterval string   `json:"poll_interval" yaml:"poll_interval"`
}

// GitRepoConfig is a git repository source.
type GitRepoConfig struct {
	URL          string `json:"url" yaml:"url"`
	Ref          string `json:"ref" yaml:"ref"`
	PollInterval string `json:"poll_interval" yaml:"poll_interval"`
}

// Source discovers and provides plugin artifacts.
type Source interface {
	Discover(ctx context.Context) ([]*plugin.Manifest, error)
	Artifact(ctx context.Context, name string) ([]byte, error)
	Watch(ctx context.Context) (<-chan SourceEvent, error)
	Close() error
}

// SourceEvent represents a plugin source change.
type SourceEvent struct {
	Name string
	Type SourceEventType
	Path string
}

// SourceEventType is the type of a source change event.
type SourceEventType string

const (
	SourceUpdated  SourceEventType = "updated"
	SourceRemoved  SourceEventType = "removed"
)

// NewSource creates a source from its configuration.
func NewSource(cfg SourceConfig) (Source, error) {
	switch cfg.Type {
	case "local":
		return NewLocalSource(cfg.Path), nil
	case "oci":
		return nil, fmt.Errorf("OCI source not yet implemented")
	case "git":
		return nil, fmt.Errorf("Git source not yet implemented")
	default:
		return nil, fmt.Errorf("unknown source type: %s", cfg.Type)
	}
}
```

- [ ] **Step 2: Implement local source in `pkg/plugin/source/local.go`**

```go
package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"flowbot.dev/pkg/plugin"
)

// LocalSource discovers plugins from a local filesystem directory.
type LocalSource struct {
	dir string
}

// NewLocalSource creates a local filesystem plugin source.
func NewLocalSource(dir string) *LocalSource {
	return &LocalSource{dir: dir}
}

// Discover scans the directory for subdirectories with plugin.yaml.
func (s *LocalSource) Discover(ctx context.Context) ([]*plugin.Manifest, error) {
	if s.dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugin dir %s: %w", s.dir, err)
	}

	var manifests []*plugin.Manifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(s.dir, entry.Name(), "plugin.yaml")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		m, err := plugin.ParseManifest(data)
		if err != nil {
			continue
		}
		// Override name with directory name for identity safety
		m.Name = entry.Name()
		manifests = append(manifests, m)
	}
	return manifests, nil
}

// Artifact returns the wasm binary or executable for a plugin.
func (s *LocalSource) Artifact(ctx context.Context, name string) ([]byte, error) {
	pluginDir := filepath.Join(s.dir, name)
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil, fmt.Errorf("read plugin dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && (entry.Name() == "plugin.wasm" || entry.Name() == "plugin-server") {
			return os.ReadFile(filepath.Join(pluginDir, entry.Name()))
		}
	}
	return nil, fmt.Errorf("no plugin artifact found in %s", pluginDir)
}

// Watch returns a channel of source events (not implemented for local, see fsnotify).
func (s *LocalSource) Watch(ctx context.Context) (<-chan SourceEvent, error) {
	return nil, fmt.Errorf("watch not implemented for local source")
}

// Close is a no-op for local source.
func (s *LocalSource) Close() error { return nil }
```

- [ ] **Step 3: Write local source tests in `pkg/plugin/source/local_test.go`**

```go
package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSourceDiscoverEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantCount int
		wantErr   string
	}{
		{
			name:   "empty directory",
			setup:  func(t *testing.T) string { return t.TempDir() },
			wantCount: 0,
		},
		{
			name:   "nonexistent directory",
			setup:  func(t *testing.T) string { return "/nonexistent/path" },
			wantCount: 0,
		},
		{
			name: "directory with plugin",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				pluginDir := filepath.Join(dir, "my-plugin")
				os.MkdirAll(pluginDir, 0755)
				os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: my-plugin
runtime: grpc
grpc:
  binary: ./server
`), 0644)
				return dir
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			src := NewLocalSource(dir)
			manifests, err := src.Discover(context.Background())
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, manifests, tt.wantCount)
		})
	}
}
```

- [ ] **Step 4: Run local source tests**

Run: `go test ./pkg/plugin/source/ -v -count=1`
Expected: Tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/plugin/source/
git commit -m "feat(plugin): add Source interface and LocalSource with directory scanning"
```

---

## Phase 7: Plugin SDK

### Task 17: Plugin SDK

**Files:**
- Create: `pkg/plugin/sdk/types.go`
- Create: `pkg/plugin/sdk/module.go`
- Create: `pkg/plugin/sdk/serve.go`

- [ ] **Step 1: Define SDK types in `pkg/plugin/sdk/types.go`**

```go
package sdk

import "encoding/json"

// Context wraps flowbot's types.Context with SDK-friendly types.
type Context struct {
	AuthContext string            `json:"auth_context"`
	UserID      string            `json:"user_id"`
	ChannelID   string            `json:"channel_id"`
	Platform    string            `json:"platform"`
	Metadata    map[string]string `json:"metadata"`
}

// MsgPayload is the message payload returned by plugin handlers.
type MsgPayload struct {
	Text string `json:"text"`
}

// Rules holds plugin-declared rulesets.
type Rules struct {
	Commands    []any `json:"commands"`
	Forms       []any `json:"forms"`
	Webservices []any `json:"webservices"`
	Webhooks    []any `json:"webhooks"`
}

// CallError represents a plugin-level error response.
type CallError struct {
	Message string `json:"error"`
	Data    any    `json:"data"`
}
```

- [ ] **Step 2: Define Module interface and ModuleBase in `pkg/plugin/sdk/module.go`**

```go
package sdk

import "encoding/json"

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

// ModuleBase provides no-op default implementations for Module.
type ModuleBase struct{}

func (ModuleBase) Init(config json.RawMessage) error                    { return nil }
func (ModuleBase) Bootstrap() error                                     { return nil }
func (ModuleBase) Command(ctx *Context, content any) (*MsgPayload, error) { return nil, fmt.Errorf("command not implemented") }
func (ModuleBase) Form(ctx *Context, values map[string]string) (*MsgPayload, error) { return nil, fmt.Errorf("form not implemented") }
func (ModuleBase) Rules() (*Rules, error)                               { return &Rules{}, nil }
func (ModuleBase) Help() (map[string][]string, error)                   { return nil, nil }
func (ModuleBase) IsReady() bool                                        { return true }
```

- [ ] **Step 3: Define serve helpers in `pkg/plugin/sdk/serve.go`**

```go
package sdk

import (
	"github.com/hashicorp/go-plugin"
)

// ServeModule starts a go-plugin server for a module plugin.
// Called from the plugin binary's main().
func ServeModule(m Module) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "FLOWBOT_PLUGIN",
			MagicCookieValue: "flowbot-plugin-v1",
		},
		Plugins: map[string]plugin.Plugin{
			"module": &ModulePlugin{impl: m},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
```

- [ ] **Step 4: Commit**

```bash
git add pkg/plugin/sdk/
git commit -m "feat(plugin): add Go SDK (Module, ModuleBase, Context, serve helpers)"
```

---

## Phase 8: Integration

### Task 18: Config integration

**Files:**
- Modify: `pkg/config/config.go`

- [ ] **Step 1: Add Plugins field to the Flowbot config struct**

Add to the main config struct in `pkg/config/config.go`:

```go
// In the main Type struct, add:
Plugins *plugin.PluginConfig `json:"plugins" yaml:"plugins" mapstructure:"plugins"`
```

Import `"flowbot.dev/pkg/plugin"` in the imports.

- [ ] **Step 2: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat(plugin): add Plugins config section to main config"
```

---

### Task 19: Hub API endpoints

**Files:**
- Create: `internal/server/hub_plugins.go`
- Create: `internal/server/hub_plugins_test.go`

- [ ] **Step 1: Implement plugin API endpoints in `internal/server/hub_plugins.go`**

```go
package server

import (
	"github.com/gofiber/fiber/v3"

	"flowbot.dev/pkg/plugin"
)

// registerPluginRoutes mounts plugin management routes under /hub/plugins.
func (s *Server) registerPluginRoutes(app *fiber.App) {
	if s.pluginManager == nil {
		return
	}

	group := app.Group("/hub/plugins")

	group.Get("/", func(c fiber.Ctx) error {
		instances := s.pluginManager.List()
		return c.JSON(instances)
	})

	group.Get("/:name/health", func(c fiber.Ctx) error {
		name := c.Params("name")
		instances := s.pluginManager.List()
		for _, inst := range instances {
			if inst.Identity == name {
				health, err := inst.Runner.Health(c.Context())
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
				}
				return c.JSON(health)
			}
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "plugin not found"})
	})

	group.Post("/load", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "manual load not implemented"})
	})

	group.Delete("/:name", func(c fiber.Ctx) error {
		name := c.Params("name")
		if err := s.pluginManager.UnloadPlugin(c.Context(), name); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "unloaded", "name": name})
	})

	group.Post("/:name/reload", func(c fiber.Ctx) error {
		name := c.Params("name")
		// For manual reload, re-discover manifest from source first
		// For now, return not implemented
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "manual reload not implemented"})
	})
}
```

- [ ] **Step 2: Write endpoint tests in `internal/server/hub_plugins_test.go`**

```go
package server

import (
	"io"
	"net/http"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flowbot.dev/pkg/plugin"
)

func TestPluginAPIDisabled(t *testing.T) {
	t.Parallel()

	app := setupTestApp(t, func(s *Server) {
		s.pluginManager = nil
	})

	tests := []struct {
		name   string
		method string
		path   string
		wantStatus int
	}{
		{
			name:   "GET /hub/plugins returns 404 when disabled",
			method: "GET",
			path:   "/hub/plugins",
			wantStatus: 404,
		},
		{
			name:   "DELETE unknown returns 404 when disabled",
			method: "DELETE",
			path:   "/hub/plugins/test",
			wantStatus: 404,
		},
		{
			name:   "GET health returns 404",
			method: "GET",
			path:   "/hub/plugins/test/health",
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestPluginAPIListEmpty(t *testing.T) {
	t.Parallel()

	app := setupTestApp(t, func(s *Server) {
		s.pluginManager = plugin.NewPluginManager(plugin.DefaultPluginConfig(), zerolog.Nop())
	})

	req, _ := http.NewRequest("GET", "/hub/plugins", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "[]")
}
```

- [ ] **Step 3: Run endpoint tests**

Run: `go test ./internal/server/ -run TestPluginAPI -v -count=1`
Expected: Tests PASS.

- [ ] **Step 4: Add pluginManager field to Server struct and call registerPluginRoutes**

In the Server struct in `internal/server/server.go`, add:
```go
pluginManager *plugin.PluginManager
```

In the route registration function, call `s.registerPluginRoutes(app)`.

- [ ] **Step 5: Commit**

```bash
git add internal/server/hub_plugins.go internal/server/hub_plugins_test.go internal/server/server.go
git commit -m "feat(plugin): add /hub/plugins API endpoints (list, health, unload)"
```

---

### Task 20: fx DI wiring

**Files:**
- Modify: `internal/modules/fx.go`

- [ ] **Step 1: Wire PluginManager lifecycle into fx**

Add to the fx options in `internal/modules/fx.go`:

```go
fx.Provide(func(log zerolog.Logger, cfg *config.Type) *plugin.PluginManager {
    mgr := plugin.NewPluginManager(cfg.Plugins, log)
    if err := mgr.Init(context.Background(), cfg.Plugins.Config); err != nil {
        log.Error().Err(err).Msg("plugin manager init failed")
    }
    return mgr
}),

fx.Invoke(func(mgr *plugin.PluginManager, lc fx.Lifecycle) {
    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            // Gracefully unload all plugins on shutdown
            for _, inst := range mgr.List() {
                mgr.UnloadPlugin(ctx, inst.Identity)
            }
            return nil
        },
    })
}),
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/fx.go
git commit -m "feat(plugin): wire PluginManager into fx DI lifecycle"
```

---

### Task 21: BDD acceptance specs

**Files:**
- Create: `internal/modules/plugin_spec_test.go`

- [ ] **Step 1: Create BDD test file in `internal/modules/plugin_spec_test.go`**

```go
package modules_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"flowbot.dev/pkg/plugin"
	"flowbot.dev/pkg/plugin/adapter"
)

var _ = Describe("Plugin System", func() {
	var runner *stubRunner

	BeforeEach(func() {
		runner = &stubRunner{}
	})

	Describe("Module Adapter", func() {
		It("responds to commands through the adapter", func() {
			runner.callResult = json.RawMessage(`{"text": "hello from plugin"}`)

			m := &plugin.Manifest{Name: "test", Runtime: plugin.RuntimeGRPC}
			a := adapter.NewModuleAdapter(m, runner)

			payload, err := a.Command(types.Context{}, "hello")
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.Text).To(Equal("hello from plugin"))
		})

		It("handles plugin errors gracefully", func() {
			runner.callError = fmt.Errorf("plugin error")

			m := &plugin.Manifest{Name: "test", Runtime: plugin.RuntimeGRPC}
			a := adapter.NewModuleAdapter(m, runner)

			_, err := a.Command(types.Context{}, "hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin error"))
		})
	})

	Describe("Plugin Manager", func() {
		It("lists empty when no plugins loaded", func() {
			mgr := plugin.NewPluginManager(plugin.DefaultPluginConfig(), zerolog.Nop())
			Expect(mgr.List()).To(BeEmpty())
		})

		It("rejects unload of unknown plugin", func() {
			mgr := plugin.NewPluginManager(plugin.DefaultPluginConfig(), zerolog.Nop())
			err := mgr.UnloadPlugin(context.Background(), "nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("rejects reload of unknown plugin", func() {
			mgr := plugin.NewPluginManager(plugin.DefaultPluginConfig(), zerolog.Nop())
			err := mgr.ReloadPlugin(context.Background(), "nonexistent", &plugin.Manifest{}, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})
```

- [ ] **Step 2: Run BDD specs**

Run: `go tool task test:specs`
Expected: Specs PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/plugin_spec_test.go
git commit -m "test(plugin): add BDD acceptance specs for plugin system"
```

---

### Task 22: Final integration check and lint

- [ ] **Step 1: Run full test suite**

Run: `go tool task test`
Expected: All tests PASS.

- [ ] **Step 2: Run lint**

Run: `go tool task lint`
Expected: No lint errors.

- [ ] **Step 3: Run format**

Run: `go tool task format`
Expected: No formatting changes.

- [ ] **Step 4: Build**

Run: `go tool task build`
Expected: Build succeeds.

- [ ] **Step 5: Commit final integration**

```bash
git add -A
git commit -m "chore(plugin): final integration fixes and lint compliance"
```
