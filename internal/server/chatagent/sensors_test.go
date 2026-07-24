package chatagent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterHooksLintSensor(t *testing.T) {
	t.Parallel()
	chatagent.LockAppConfigForTest(t)

	prevSensors := config.App.ChatAgent.Sensors
	prevWS := config.App.ChatAgent.Workspace
	t.Cleanup(func() {
		config.App.ChatAgent.Sensors = prevSensors
		config.App.ChatAgent.Workspace = prevWS
	})
	config.App.ChatAgent.Workspace = t.TempDir()

	tests := []struct {
		name        string
		lintOnWrite bool
		toolName    string
		args        map[string]any
		wantParts   bool
	}{
		{
			name:        "go write observed without rewrite",
			lintOnWrite: true,
			toolName:    "write_file",
			args:        map[string]any{"path": "main.go"},
			wantParts:   false,
		},
		{
			name:        "non-go ignored",
			lintOnWrite: true,
			toolName:    "write_file",
			args:        map[string]any{"path": "readme.md"},
			wantParts:   false,
		},
		{
			name:        "disabled sensor no-op",
			lintOnWrite: false,
			toolName:    "write_file",
			args:        map[string]any{"path": "main.go"},
			wantParts:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.Sensors.LintOnWrite = tt.lintOnWrite
			reg := hooks.NewRegistry()
			chatagent.RegisterHooks(reg, chatagent.ChatHookDeps{SessionID: "lint-test", Service: chatagent.NewService()})
			cfg := hooks.BridgeConfig(context.Background(), reg, msg.Config{})
			require.NotNil(t, cfg.AfterToolCall)
			result, err := cfg.AfterToolCall(msg.AfterToolContext{
				ToolCall: msg.ToolCallPart{Name: tt.toolName, ID: "1"},
				Args:     tt.args,
				Result: msg.ToolResultMessage{
					ToolCallID: "1",
					Name:       tt.toolName,
					Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
				},
			})
			require.NoError(t, err)
			if !tt.wantParts {
				if result != nil {
					assert.Empty(t, result.Parts)
					assert.Nil(t, result.IsError)
				}
			}
		})
	}
}

func TestRegisterHooksPathSensor(t *testing.T) {
	t.Parallel()
	chatagent.LockAppConfigForTest(t)

	prev := config.App.ChatAgent.Workspace
	t.Cleanup(func() { config.App.ChatAgent.Workspace = prev })
	config.App.ChatAgent.Workspace = t.TempDir()

	tests := []struct {
		name      string
		toolName  string
		args      map[string]any
		wantError bool
	}{
		{name: "inside workspace", toolName: "write_file", args: map[string]any{"path": "ok.go"}, wantError: false},
		{name: "escape path", toolName: "write_file", args: map[string]any{"path": "../../etc/passwd"}, wantError: true},
		{name: "list_dir default path ok", toolName: "list_dir", args: map[string]any{}, wantError: false},
		{name: "glob_files nil path ok", toolName: "glob_files", args: map[string]any{"pattern": "*.go"}, wantError: false},
		{name: "grep_files missing path ok", toolName: "grep_files", args: map[string]any{"pattern": "x"}, wantError: false},
		{name: "run_code default filename inside", toolName: "run_code", args: map[string]any{"language": "python", "code": "print(1)"}, wantError: false},
		{name: "run_code escape filename", toolName: "run_code", args: map[string]any{"language": "python", "code": "x", "filename": "../../etc/passwd.py"}, wantError: true},
		{name: "other tool ignored", toolName: "echo", args: map[string]any{"path": "../../x"}, wantError: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			chatagent.RegisterHooks(reg, chatagent.ChatHookDeps{SessionID: "s1"})
			cfg := hooks.BridgeConfig(context.Background(), reg, msg.Config{})
			require.NotNil(t, cfg.AfterToolCall)
			result, err := cfg.AfterToolCall(msg.AfterToolContext{
				ToolCall: msg.ToolCallPart{Name: tt.toolName, ID: "1"},
				Args:     tt.args,
				Result:   msg.ToolResultMessage{ToolCallID: "1", Name: tt.toolName, Parts: []msg.ContentPart{msg.TextPart{Text: "ok"}}},
			})
			require.NoError(t, err)
			if !tt.wantError {
				if result != nil && result.IsError != nil {
					assert.False(t, *result.IsError)
				}
				return
			}
			require.NotNil(t, result)
			require.NotNil(t, result.IsError)
			assert.True(t, *result.IsError)
		})
	}
}
