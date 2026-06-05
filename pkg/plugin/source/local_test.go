package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSourceDiscover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantCount int
	}{
		{
			name:      "empty directory",
			setup:     func(t *testing.T) string { return t.TempDir() },
			wantCount: 0,
		},
		{
			name:      "nonexistent directory",
			setup:     func(_ *testing.T) string { return "/nonexistent/path" },
			wantCount: 0,
		},
		{
			name: "directory with plugin",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				pluginDir := filepath.Join(dir, "my-plugin")
				_ = os.MkdirAll(pluginDir, 0755)
				_ = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte("name: my-plugin\nversion: \"1.0\"\nruntime: grpc\ngrpc:\n  binary: ./server\n"), 0644)
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
			require.NoError(t, err)
			assert.Len(t, manifests, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, "my-plugin", manifests[0].Name)
			}
		})
	}
}
