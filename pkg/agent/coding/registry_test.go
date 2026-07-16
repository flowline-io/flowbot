package coding_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveToolNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns five default tools"},
		{name: "includes run_terminal"},
		{name: "includes web_search"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			names := coding.ActiveToolNames()
			require.Len(t, names, 5)
			assert.Contains(t, names, "run_terminal")
			assert.Contains(t, names, "read_file")
			assert.Contains(t, names, "write_file")
			assert.Contains(t, names, "web_search")
			assert.Contains(t, names, "run_code")
		})
	}
}

func TestRegisterAll(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ws := coding.Workspace{Root: root}

	tests := []struct {
		name    string
		env     env.ExecutionEnv
		wantErr bool
	}{
		{name: "registers all tools with default env", env: nil},
		{name: "registers all tools with explicit env", env: env.Default()},
		{name: "duplicate registration fails", env: env.Default(), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			registry := tool.NewRegistry()
			err := coding.RegisterAll(registry, ws, tt.env)
			require.NoError(t, err)
			if !tt.wantErr {
				for _, name := range coding.ActiveToolNames() {
					_, ok := registry.Get(name)
					assert.True(t, ok, "missing tool %s", name)
				}
				return
			}
			err = coding.RegisterAll(registry, ws, tt.env)
			require.Error(t, err)
		})
	}
}
