package source

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/plugin"
	plugintypes "github.com/flowline-io/flowbot/pkg/plugin/types"
)

// SourceConfig is an alias for plugintypes.SourceConfig.
type SourceConfig = plugintypes.SourceConfig

// GitRepoConfig is an alias for plugintypes.GitRepoConfig.
type GitRepoConfig = plugintypes.GitRepoConfig

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
	SourceUpdated SourceEventType = "updated"
	SourceRemoved SourceEventType = "removed"
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
