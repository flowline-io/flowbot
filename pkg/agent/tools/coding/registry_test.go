package coding_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveToolNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns ten default tools"},
		{name: "includes run_terminal"},
		{name: "includes web_fetch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			names := coding.ActiveToolNames()
			require.Len(t, names, 10)
			assert.Contains(t, names, "run_terminal")
			assert.Contains(t, names, "list_dir")
			assert.Contains(t, names, "glob_files")
			assert.Contains(t, names, "grep_files")
			assert.Contains(t, names, "read_file")
			assert.Contains(t, names, "write_file")
			assert.Contains(t, names, "apply_patch")
			assert.Contains(t, names, "web_search")
			assert.Contains(t, names, "web_fetch")
			assert.Contains(t, names, "run_code")
		})
	}
}

func TestRegisterHeadless(t *testing.T) {
	t.Parallel()
	ws := coding.Workspace{Root: t.TempDir()}
	reg := tool.NewRegistry()
	names, err := coding.RegisterHeadless(reg, ws, nil, coding.HeadlessOptions{Force: true})
	require.NoError(t, err)
	assert.Equal(t, coding.HeadlessToolNames(true), names)
	for _, name := range names {
		_, ok := reg.Get(name)
		assert.True(t, ok, "missing %s", name)
	}
	_, ok := reg.Get("web_search")
	assert.False(t, ok)

	regRO := tool.NewRegistry()
	roNames, err := coding.RegisterHeadless(regRO, ws, nil, coding.HeadlessOptions{Force: false})
	require.NoError(t, err)
	assert.Equal(t, coding.HeadlessToolNames(false), roNames)
	_, ok = regRO.Get("write_file")
	assert.False(t, ok)
	_, ok = regRO.Get("run_terminal")
	assert.False(t, ok)
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
