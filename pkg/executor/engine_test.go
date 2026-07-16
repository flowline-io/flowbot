package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEngineState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(e *Engine) string
		want  string
	}{
		{name: "new engine is idle", setup: func(_ *Engine) string { return "" }, want: "IDLE"},
		{name: "closed engine reports closed", setup: func(e *Engine) string {
			_ = e.Close()
			return ""
		}, want: "CLOSED"},
		{name: "double close stays closed", setup: func(e *Engine) string {
			_ = e.Close()
			_ = e.Close()
			return ""
		}, want: "CLOSED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New(runtime.Capability)
			tt.setup(e)
			assert.Equal(t, tt.want, e.State())
		})
	}
}

func TestEngineClose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "close idle engine succeeds"},
		{name: "close twice is idempotent"},
		{name: "close without runtime succeeds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New(runtime.Capability)
			require.NoError(t, e.Close())
			require.NoError(t, e.Close())
			assert.Equal(t, "CLOSED", e.State())
		})
	}
}

func TestEngineRunValidation(t *testing.T) {
	orig := config.App
	t.Cleanup(func() { config.App = orig })
	config.App.Executor = config.Executor{
		Limits: config.ExecutorLimits{Cpus: "1", Memory: "256m"},
	}

	tests := []struct {
		name    string
		setup   func(e *Engine)
		task    *types.Task
		wantErr string
	}{
		{
			name:    "closed engine rejects run",
			setup:   func(e *Engine) { _ = e.Close() },
			task:    &types.Task{Run: "capability:example.list"},
			wantErr: "not idle",
		},
		{
			name:    "invalid timeout duration",
			task:    &types.Task{Run: "capability:example.list", Timeout: "not-a-duration"},
			wantErr: "invalid timeout duration",
		},
		{
			name:    "invalid capability action format",
			task:    &types.Task{Run: "capability:invalid"},
			wantErr: "invalid capability action",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New(runtime.Capability)
			if tt.setup != nil {
				tt.setup(e)
			}
			err := e.Run(context.Background(), tt.task)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestEngineRunAppliesDefaultLimits(t *testing.T) {
	orig := config.App
	t.Cleanup(func() { config.App = orig })
	config.App.Executor = config.Executor{
		Limits: config.ExecutorLimits{Cpus: "2", Memory: "512m"},
	}

	e := New(runtime.Capability)
	task := &types.Task{Run: "capability:missing.op"}
	_ = e.Run(context.Background(), task)
	require.NotNil(t, task.Limits)
	assert.Equal(t, "2", task.Limits.CPUs)
	assert.Equal(t, "512m", task.Limits.Memory)
}

func TestEngineUnknownRuntime(t *testing.T) {
	t.Parallel()
	e := New("unknown-runtime")
	err := e.Run(context.Background(), &types.Task{Run: "echo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown runtime type")
}
