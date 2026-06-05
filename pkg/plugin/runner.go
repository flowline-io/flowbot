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
