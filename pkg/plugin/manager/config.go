// Package manager provides the PluginManager that orchestrates plugin
// discovery, loading, lifecycle, and hot-reload.
package manager

import (
	"encoding/json"
	"time"
)

// SourceConfig defines a plugin distribution source entry.
// Note: When the source package (Task 16) is implemented, this type
// should move to pkg/plugin/source/source.go.
type SourceConfig struct {
	Type         string `json:"type" yaml:"type"`
	Path         string `json:"path" yaml:"path"`
	Registry     string `json:"registry" yaml:"registry"`
	PollInterval string `json:"poll_interval" yaml:"poll_interval"`
}

// PluginConfig is the top-level plugins section in flowbot.yaml.
type PluginConfig struct {
	Enabled      bool                       `json:"enabled" yaml:"enabled"`
	Sources      []SourceConfig             `json:"sources" yaml:"sources"`
	Config       map[string]json.RawMessage `json:"config" yaml:"config"`
	HotReload    bool                       `json:"hot_reload" yaml:"hot_reload"`
	DrainTimeout time.Duration              `json:"drain_timeout" yaml:"drain_timeout"`
	MaxPlugins   int                        `json:"max_plugins" yaml:"max_plugins"`
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
