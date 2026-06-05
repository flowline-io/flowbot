package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	goYaml "github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Manifest is the parsed plugin.yaml configuration.
type Manifest struct {
	Name         string          `json:"name" yaml:"name"`
	Version      string          `json:"version" yaml:"version"`
	Description  string          `json:"description" yaml:"description"`
	Author       string          `json:"author" yaml:"author"`
	Runtime      RuntimeKind     `json:"runtime" yaml:"runtime"`
	Provides     Provides        `json:"provides" yaml:"provides"`
	GRPC         *GRPCConfig     `json:"grpc" yaml:"grpc"`
	Wasm         *WasmConfig     `json:"wasm" yaml:"wasm"`
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
	Module    bool          `json:"module" yaml:"module"`
	Abilities []AbilityDecl `json:"abilities" yaml:"abilities"`
	Provider  *ProviderDecl `json:"provider" yaml:"provider"`
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
	HTTP       []HTTPPermission `json:"http" yaml:"http"`
	Filesystem []FSPermission   `json:"filesystem" yaml:"filesystem"`
	Memory     *MemoryLimit     `json:"memory" yaml:"memory"`
	Execution  *ExecutionLimit  `json:"execution" yaml:"execution"`
}

// HTTPPermission allowlist entry for HTTP requests from Wasm.
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
	if err := goYaml.Unmarshal(data, &m); err != nil {
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
// Returns nil if no schema is defined.
func (m *Manifest) ValidateConfig(config json.RawMessage) error {
	if len(m.ConfigSchema) == 0 {
		return nil
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(m.ConfigSchema)))
	if err != nil {
		return fmt.Errorf("manifest config_schema is invalid: %w", err)
	}
	var v any
	if err := sonic.Unmarshal(config, &v); err != nil {
		return fmt.Errorf("plugin config is not valid JSON: %w", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("plugin.yaml", schemaDoc); err != nil {
		return fmt.Errorf("manifest config_schema is invalid: %w", err)
	}
	sch, err := c.Compile("plugin.yaml")
	if err != nil {
		return fmt.Errorf("manifest config_schema is invalid: %w", err)
	}
	if err := sch.Validate(v); err != nil {
		return fmt.Errorf("plugin config validation failed: %w", err)
	}
	return nil
}
