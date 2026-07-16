package hooks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmitBeforeAgentStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*hooks.Registry)
		wantCancel bool
		wantPrompt string
		wantErr    bool
	}{
		{
			name: "cancel flag merges",
			setup: func(reg *hooks.Registry) {
				hooks.OnBeforeAgentStart(reg, func(_ context.Context, _ hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
					return &hooks.BeforeAgentStartResult{Cancel: true}, nil
				})
			},
			wantCancel: true,
		},
		{
			name: "system prompt replacement merges",
			setup: func(reg *hooks.Registry) {
				prompt := "updated"
				hooks.OnBeforeAgentStart(reg, func(_ context.Context, _ hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
					return &hooks.BeforeAgentStartResult{SystemPrompt: &prompt}, nil
				})
			},
			wantPrompt: "updated",
		},
		{
			name: "handler error propagates",
			setup: func(reg *hooks.Registry) {
				hooks.OnBeforeAgentStart(reg, func(context.Context, hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
					return nil, errors.New("start failed")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			result, err := reg.EmitBeforeAgentStart(context.Background(), hooks.BeforeAgentStartEvent{
				Messages:     []msg.AgentMessage{userMessage("hi")},
				SystemPrompt: "base",
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.wantCancel {
				assert.True(t, result.Cancel)
			}
			if tt.wantPrompt != "" {
				require.NotNil(t, result.SystemPrompt)
				assert.Equal(t, tt.wantPrompt, *result.SystemPrompt)
			}
		})
	}
}

func TestEmitToolResultMergesPatches(t *testing.T) {
	t.Parallel()

	isErr := true

	tests := []struct {
		name          string
		setup         func(*hooks.Registry)
		wantParts     int
		wantError     bool
		wantTerminate bool
		wantErr       bool
	}{
		{
			name: "merges parts from handlers",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolResult(reg, func(_ context.Context, _ hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
					return &hooks.ToolResultResult{Parts: []msg.ContentPart{msg.TextPart{Text: "patched"}}}, nil
				})
			},
			wantParts: 1,
		},
		{
			name: "merges terminate flag",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolResult(reg, func(_ context.Context, _ hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
					return &hooks.ToolResultResult{Terminate: true}, nil
				})
			},
			wantTerminate: true,
		},
		{
			name: "handler error propagates",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolResult(reg, func(context.Context, hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
					return nil, errors.New("result failed")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			if tt.name == "merges parts from handlers" {
				hooks.OnToolResult(reg, func(_ context.Context, _ hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
					return &hooks.ToolResultResult{IsError: &isErr}, nil
				})
			}
			result, err := reg.EmitToolResult(context.Background(), hooks.ToolResultEvent{
				Result: msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "raw"}}},
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.wantParts > 0 {
				assert.Len(t, result.Parts, tt.wantParts)
			}
			if tt.wantError {
				require.NotNil(t, result.IsError)
				assert.True(t, *result.IsError)
			}
			if tt.wantTerminate {
				assert.True(t, result.Terminate)
			}
		})
	}
}

func TestOnObservationFiltersByType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		eventType string
		wantCalls int
	}{
		{name: "matching type invokes handler", eventType: hooks.EventSavePoint, wantCalls: 1},
		{name: "other type is ignored", eventType: hooks.EventContextUsage, wantCalls: 0},
		{name: "empty type ignored", eventType: "", wantCalls: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			calls := 0
			hooks.OnObservation(reg, hooks.EventSavePoint, func(context.Context, hooks.ObservationEvent) error {
				calls++
				return nil
			})
			reg.EmitObservation(context.Background(), hooks.ObservationEvent{Type: tt.eventType}, nil)
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestBridgeConfigToolHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(*hooks.Registry)
		wantBlock   bool
		wantPatched bool
	}{
		{
			name: "tool call hook blocks via bridge",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolCall(reg, func(_ context.Context, _ hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
					return &hooks.ToolCallResult{Block: true, Reason: "denied"}, nil
				})
			},
			wantBlock: true,
		},
		{
			name: "tool result hook patches via bridge",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolResult(reg, func(_ context.Context, _ hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
					return &hooks.ToolResultResult{Parts: []msg.ContentPart{msg.TextPart{Text: "hook"}}}, nil
				})
			},
			wantPatched: true,
		},
		{
			name: "before agent start registers loop handler",
			setup: func(reg *hooks.Registry) {
				hooks.OnBeforeAgentStart(reg, func(context.Context, hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
					return nil, nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			cfg := hooks.BridgeConfig(context.Background(), reg, msg.Config{})
			if tt.wantBlock {
				require.NotNil(t, cfg.BeforeToolCall)
				result, err := cfg.BeforeToolCall(msg.BeforeToolContext{})
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.Block)
				return
			}
			if tt.wantPatched {
				require.NotNil(t, cfg.AfterToolCall)
				result, err := cfg.AfterToolCall(msg.AfterToolContext{
					Result: msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "base"}}},
				})
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Parts)
				return
			}
			assert.True(t, reg.HasLoopHandlers())
		})
	}
}
