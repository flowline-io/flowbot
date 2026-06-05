// Package manager provides the PluginManager that orchestrates plugin
// discovery, loading, lifecycle, and hot-reload.
package manager

import (
	plugintypes "github.com/flowline-io/flowbot/pkg/plugin/types"
)

// PluginConfig is a type alias for plugintypes.PluginConfig.
type PluginConfig = plugintypes.PluginConfig

// SourceConfig is a type alias for plugintypes.SourceConfig.
type SourceConfig = plugintypes.SourceConfig

// DefaultPluginConfig calls plugintypes.DefaultPluginConfig to maintain backward
// compatibility with existing callers.
func DefaultPluginConfig() *PluginConfig {
	return plugintypes.DefaultPluginConfig()
}
