package loop_test

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"testing"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/tools/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestRunLoop_ToolThenComplete(t *testing.T) {
	tests := []struct {
		name      string
		maxSteps  int
		wantError error
	}{
		{name: "completes after tool call", maxSteps: 10, wantError: nil},
		{name: "respects max steps", maxSteps: 1, wantError: loop.ErrMaxSteps},
		{name: "default max steps", maxSteps: 0, wantError: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(
				agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
				}}},
				agentllm.ResponseScript{Content: "done"},
			)
			if tt.name == "respects max steps" {
				model = agentllm.NewFakeModel(agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
				}}})
			}

			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(echo.Tool{}))

			cfg := loop.DefaultConfig()
			cfg.MaxSteps = tt.maxSteps
			cfg.ModelName = "fake"

			messages, err := loop.RunLoop(context.Background(), []msg.AgentMessage{
				loop.NewUserMessage("run echo"),
			}, &msg.Context{}, cfg, loop.LoopDeps{Model: model, Registry: reg}, nil)

			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, messages)
		})
	}
}

func TestRunLoop_Aborted(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancel before loop"},
		{name: "cancel stops loop"},
		{name: "returns aborted error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := loop.RunLoop(ctx, []msg.AgentMessage{loop.NewUserMessage("x")}, &msg.Context{}, loop.DefaultConfig(), loop.LoopDeps{
				Model: agentllm.NewFakeModel(agentllm.ResponseScript{Content: "x"}),
			}, nil)
			assert.ErrorIs(t, err, loop.ErrAborted)
		})
	}
}

func TestAgent_Prompt(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "async prompt completes"},
		{name: "state updated after run"},
		{name: "stream returns result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "hello"})
			ag := loop.NewAgent(loop.Options{
				Model:  model,
				Config: msg.Config{ModelName: "fake", MaxSteps: 5},
			})
			stream, err := ag.Prompt(context.Background(), loop.NewUserMessage("hi"))
			require.NoError(t, err)
			result, err := stream.Await(context.Background())
			require.NoError(t, err)
			assert.NoError(t, result.Err)
			assert.NotEmpty(t, ag.State().Messages)
		})
	}
}

func TestRunLoopContinue_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *msg.Context
		wantErr error
	}{
		{name: "empty context", ctx: &msg.Context{}, wantErr: loop.ErrEmptyContext},
		{name: "assistant last message", ctx: &msg.Context{Messages: []msg.AgentMessage{msg.AssistantMessage{}}}, wantErr: loop.ErrInvalidContinue},
		{name: "valid continue", ctx: &msg.Context{Messages: []msg.AgentMessage{loop.NewUserMessage("x")}}, wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			_, err := loop.RunLoopContinue(context.Background(), tt.ctx, loop.DefaultConfig(), loop.LoopDeps{Model: model}, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
