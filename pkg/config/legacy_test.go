package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRejectLegacyKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings map[string]any
		wantErr  string
		noErr    bool
	}{
		{
			name: "clean modern config",
			settings: map[string]any{
				"postgres": map[string]any{"dsn": "postgres://x"},
				"redis":    map[string]any{"url": "redis://:pass@127.0.0.1:6379/0"},
			},
			noErr: true,
		},
		{
			name:     "store_config present",
			settings: map[string]any{"store_config": map[string]any{"use_adapter": "postgres"}},
			wantErr:  "store_config: removed",
		},
		{
			name: "redis host present",
			settings: map[string]any{
				"redis": map[string]any{"host": "127.0.0.1", "url": "redis://:p@127.0.0.1:6379/0"},
			},
			wantErr: "redis.host: removed",
		},
		{
			name: "redis password present",
			settings: map[string]any{
				"redis": map[string]any{"password": "secret"},
			},
			wantErr: "redis.password: removed",
		},
		{
			name: "redis port present",
			settings: map[string]any{
				"redis": map[string]any{"port": 6379},
			},
			wantErr: "redis.port: removed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := RejectLegacyKeys(tt.settings)
			if tt.noErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), "Fix:")
		})
	}
}
