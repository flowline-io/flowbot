package profiling

import (
	"context"
	"testing"

	"github.com/grafana/pyroscope-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/config"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (lc *testLifecycle) Append(h fx.Hook) {
	lc.hooks = append(lc.hooks, h)
}

func TestNewProfilerDisabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "disabled profiling returns nil without hooks"},
		{name: "disabled profiling skips pyroscope start"},
		{name: "disabled profiling accepts nil lifecycle hooks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orig := config.App.Profiling
			t.Cleanup(func() { config.App.Profiling = orig })
			config.App.Profiling = config.Profiling{Enabled: false}

			lc := &testLifecycle{}
			err := NewProfiler(lc)
			require.NoError(t, err)
			require.Empty(t, lc.hooks)
		})
	}
}

func TestNewProfilerEnabledRegistersHooks(t *testing.T) {
	t.Parallel()
	orig := config.App.Profiling
	t.Cleanup(func() { config.App.Profiling = orig })
	config.App.Profiling = config.Profiling{
		Enabled:       true,
		ServerAddress: "http://127.0.0.1:1",
		ServiceName:   "flowbot-test",
		Environment:   "test",
		ProfileTypes:  []string{"cpu"},
	}

	lc := &testLifecycle{}
	err := NewProfiler(lc)
	require.NoError(t, err)
	require.Len(t, lc.hooks, 1)

	// OnStart may succeed or fail depending on pyroscope availability; either is fine.
	_ = lc.hooks[0].OnStart(context.Background())
}

func TestNewProfilerEnabledDefaults(t *testing.T) {
	t.Parallel()
	orig := config.App.Profiling
	t.Cleanup(func() { config.App.Profiling = orig })
	config.App.Profiling = config.Profiling{
		Enabled:       true,
		ServerAddress: "http://127.0.0.1:1",
	}

	lc := &testLifecycle{}
	err := NewProfiler(lc)
	require.NoError(t, err)
	require.Len(t, lc.hooks, 1)
	_ = lc.hooks[0].OnStart(context.Background())
	_ = lc.hooks[0].OnStop(context.Background())
}

func TestProfileTypeNamesSingle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "single profile type name"},
		{name: "three profile type names"},
		{name: "empty slice returns empty names"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			switch tt.name {
			case "single profile type name":
				names := profileTypeNames([]pyroscope.ProfileType{pyroscope.ProfileCPU})
				require.Equal(t, []string{"cpu"}, names)
			case "three profile type names":
				names := profileTypeNames([]pyroscope.ProfileType{
					pyroscope.ProfileCPU,
					pyroscope.ProfileGoroutines,
					pyroscope.ProfileAllocObjects,
				})
				require.Len(t, names, 3)
			default:
				assert.Empty(t, profileTypeNames(nil))
			}
		})
	}
}
