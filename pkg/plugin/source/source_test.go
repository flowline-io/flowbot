package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cfg       SourceConfig
		wantErr   string
		wantLocal bool
	}{
		{name: "local source", cfg: SourceConfig{Type: "local", Path: "/tmp"}, wantLocal: true},
		{name: "oci not implemented", cfg: SourceConfig{Type: "oci"}, wantErr: "OCI source not yet implemented"},
		{name: "git not implemented", cfg: SourceConfig{Type: "git"}, wantErr: "Git source not yet implemented"},
		{name: "unknown type", cfg: SourceConfig{Type: "ftp"}, wantErr: "unknown source type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src, err := NewSource(tt.cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantLocal {
				_, ok := src.(*LocalSource)
				assert.True(t, ok)
			}
		})
	}
}

func TestLocalSourceArtifactAndWatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "artifact missing plugin returns error"},
		{name: "watch returns not implemented"},
		{name: "close is no-op"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := NewLocalSource(t.TempDir())
			switch tt.name {
			case "artifact missing plugin returns error":
				_, err := src.Artifact(t.Context(), "missing")
				require.Error(t, err)
			case "watch returns not implemented":
				_, err := src.Watch(t.Context())
				require.Error(t, err)
			case "close is no-op":
				require.NoError(t, src.Close())
			}
		})
	}
}

func TestLocalSourceArtifactSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		content  []byte
	}{
		{name: "reads plugin wasm artifact", filename: "plugin.wasm", content: []byte{0, 1, 2}},
		{name: "reads plugin-server artifact", filename: "plugin-server", content: []byte("binary")},
		{name: "reads wasm bytes exactly", filename: "plugin.wasm", content: []byte("wasm")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			pluginDir := filepath.Join(dir, "demo-plugin")
			require.NoError(t, os.MkdirAll(pluginDir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(pluginDir, tt.filename), tt.content, 0o644))

			src := NewLocalSource(dir)
			got, err := src.Artifact(context.Background(), "demo-plugin")
			require.NoError(t, err)
			assert.Equal(t, tt.content, got)
		})
	}
}

func TestLocalSourceDiscoverEmptyPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "empty path returns nil manifests"},
		{name: "empty path does not error"},
		{name: "empty path has no plugins"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := NewLocalSource("")
			manifests, err := src.Discover(context.Background())
			require.NoError(t, err)
			assert.Nil(t, manifests)
		})
	}
}
