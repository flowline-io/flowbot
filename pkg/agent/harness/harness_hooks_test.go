package harness_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestHarnessBeforeAgentStartMutatesSystemPrompt(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*hooks.Registry)
		wantPrompt string
		wantErr    bool
	}{
		{
			name: "hook updates system prompt",
			setup: func(reg *hooks.Registry) {
				hooks.OnBeforeAgentStart(reg, func(_ context.Context, _ hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
					prompt := "hooked prompt"
					return &hooks.BeforeAgentStartResult{SystemPrompt: &prompt}, nil
				})
			},
			wantPrompt: "hooked prompt",
		},
		{
			name: "cancel aborts prompt",
			setup: func(reg *hooks.Registry) {
				hooks.OnBeforeAgentStart(reg, func(_ context.Context, _ hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
					return &hooks.BeforeAgentStartResult{Cancel: true}, nil
				})
			},
			wantErr: true,
		},
		{
			name:       "no hook keeps original prompt",
			setup:      func(_ *hooks.Registry) {},
			wantPrompt: "base prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			h := harness.New(harness.Options{
				AgentOptions: agent.Options{Model: fakeModel},
				SystemPrompt: "base prompt",
				ModelName:    "fake",
				Hooks:        reg,
			})

			stream, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, hooks.ErrRunCancelled)
				return
			}
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			require.NotNil(t, stream)
			assert.Equal(t, tt.wantPrompt, h.Agent().State().SystemPrompt)
		})
	}
}

func TestHarnessContextHookRunsDuringPrompt(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*hooks.Registry, *atomic.Int32)
		wantCalls int32
	}{
		{
			name: "context hook invoked once per run",
			setup: func(reg *hooks.Registry, calls *atomic.Int32) {
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					calls.Add(1)
					return &hooks.ContextResult{Messages: ev.Messages}, nil
				})
			},
			wantCalls: 1,
		},
		{
			name: "multiple context hooks chain",
			setup: func(reg *hooks.Registry, calls *atomic.Int32) {
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					calls.Add(1)
					return &hooks.ContextResult{Messages: ev.Messages}, nil
				})
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					calls.Add(1)
					return &hooks.ContextResult{Messages: ev.Messages}, nil
				})
			},
			wantCalls: 2,
		},
		{
			name:      "no context hook leaves counter zero",
			setup:     func(_ *hooks.Registry, _ *atomic.Int32) {},
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			reg := hooks.NewRegistry()
			var calls atomic.Int32
			tt.setup(reg, &calls)
			fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			h := harness.New(harness.Options{
				AgentOptions: agent.Options{Model: fakeModel},
				ModelName:    "fake",
				Hooks:        reg,
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}

func TestHarnessRepeatedPromptDoesNotDoubleWrapHooks(t *testing.T) {
	tests := []struct {
		name      string
		prompts   int
		wantCalls int32
	}{
		{name: "single prompt invokes context hook once", prompts: 1, wantCalls: 1},
		{name: "two prompts invoke context hook twice total", prompts: 2, wantCalls: 2},
		{name: "three prompts invoke context hook thrice total", prompts: 3, wantCalls: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			reg := hooks.NewRegistry()
			var calls atomic.Int32
			hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
				calls.Add(1)
				return &hooks.ContextResult{Messages: ev.Messages}, nil
			})
			fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			h := harness.New(harness.Options{
				AgentOptions: agent.Options{Model: fakeModel},
				ModelName:    "fake",
				Hooks:        reg,
			})

			for range tt.prompts {
				_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
				require.NoError(t, err)
				require.NoError(t, h.WaitIdle(ctx))
			}
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}

func TestHarnessToolResultTerminateStopsLoop(t *testing.T) {
	tests := []struct {
		name      string
		terminate bool
		wantCalls int
	}{
		{name: "terminate after tool skips second model call", terminate: true, wantCalls: 1},
		{name: "without terminate continues to assistant reply", terminate: false, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			reg := hooks.NewRegistry()
			if tt.terminate {
				hooks.OnToolResult(reg, func(_ context.Context, _ hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
					return &hooks.ToolResultResult{Terminate: true}, nil
				})
			}
			fakeModel := agentllm.NewFakeModel(
				agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
				}}},
				agentllm.ResponseScript{Content: "done"},
			)
			toolRegistry := tool.NewRegistry()
			require.NoError(t, toolRegistry.Register(echo.Tool{}))
			h := harness.New(harness.Options{
				AgentOptions: agent.Options{
					Model:    fakeModel,
					Registry: toolRegistry,
					Config:   agent.Config{MaxSteps: 5},
				},
				ModelName: "fake",
				Hooks:     reg,
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("run"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			require.NoError(t, h.LastRunResult().Err)
			assert.Equal(t, tt.wantCalls, fakeModel.Calls())
		})
	}
}

func TestHarnessBridgeRespectsCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reg := hooks.NewRegistry()
	hooks.OnContext(reg, func(callCtx context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
		if errors.Is(callCtx.Err(), context.Canceled) {
			return nil, context.Canceled
		}
		return &hooks.ContextResult{Messages: ev.Messages}, nil
	})
	fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
	h := harness.New(harness.Options{
		AgentOptions: agent.Options{Model: fakeModel},
		ModelName:    "fake",
		Hooks:        reg,
	})

	_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, h.WaitIdle(context.Background()))
	require.Error(t, h.LastRunResult().Err)
	assert.ErrorIs(t, h.LastRunResult().Err, context.Canceled)
}
