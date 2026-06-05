// Package source provides plugin discovery from local filesystem directories.
package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flowline-io/flowbot/pkg/plugin"
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
func (s *LocalSource) Discover(_ context.Context) ([]*plugin.Manifest, error) {
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
		m.Name = entry.Name()
		manifests = append(manifests, m)
	}
	return manifests, nil
}

// Artifact returns the wasm binary or executable for a plugin.
func (s *LocalSource) Artifact(_ context.Context, name string) ([]byte, error) {
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

func (*LocalSource) Watch(_ context.Context) (<-chan SourceEvent, error) {
	return nil, fmt.Errorf("watch not implemented for local source")
}

func (*LocalSource) Close() error { return nil }
