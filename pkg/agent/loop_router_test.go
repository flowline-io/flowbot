package agent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func echoToolCall(id string) llms.ToolCall {
	return llms.ToolCall{
		ID:   id,
		Type: "function",
		FunctionCall: &llms.FunctionCall{
			Name:      "echo",
			Arguments: `{"text":"hi"}`,
		},
	}
}

func assistantModels(messages []agent.AgentMessage) []string {
	models := make([]string, 0)
	for _, message := range messages {
		assistant, ok := message.(agent.AssistantMessage)
		if !ok {
			continue
		}
		models = append(models, assistant.Model)
	}
	return models
}

func TestRunLoop_DualModelRouting(t *testing.T) {
	tests := []struct {
		name      string
		dual      bool
		scripts   []agentllm.ResponseScript
		wantModel []string
	}{
		{
			name: "switches to tool model after first tool round",
			dual: true,
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{echoToolCall("call-1")}},
				{Content: "done"},
			},
			wantModel: []string{"chat-model", "tool-model"},
		},
		{
			name: "keeps tool model across chained tool rounds",
			dual: true,
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{echoToolCall("call-1")}},
				{ToolCalls: []llms.ToolCall{echoToolCall("call-2")}},
				{Content: "done"},
			},
			wantModel: []string{"chat-model", "tool-model", "tool-model"},
		},
		{
			name: "single model when dual fields unset",
			dual: false,
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{echoToolCall("call-1")}},
				{Content: "done"},
			},
			wantModel: []string{"only-model", "only-model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(tt.scripts...)
			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(echo.Tool{}))

			cfg := agent.DefaultConfig()
			cfg.MaxSteps = 10
			if tt.dual {
				cfg.ChatModel = "chat-model"
				cfg.ToolModel = "tool-model"
				cfg.ModelName = "chat-model"
			} else {
				cfg.ModelName = "only-model"
			}

			messages, err := agent.RunLoop(context.Background(), []agent.AgentMessage{
				agent.NewUserMessage("run echo"),
			}, &agent.Context{}, cfg, agent.LoopDeps{Model: model, Registry: reg}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantModel, assistantModels(messages))
		})
	}
}

func TestRunLoop_DualModelResolvesPerTurnClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		wantChatCalls int
		wantToolCalls int
	}{
		{
			name:          "tool turn uses tool-model client not chat client",
			wantChatCalls: 1,
			wantToolCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chatModel := agentllm.NewFakeModel(agentllm.ResponseScript{
				ToolCalls: []llms.ToolCall{echoToolCall("call-1")},
			})
			toolModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "done"})
			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(echo.Tool{}))

			cfg := agent.DefaultConfig()
			cfg.MaxSteps = 10
			cfg.ChatModel = "chat-model"
			cfg.ToolModel = "tool-model"
			cfg.ModelName = "chat-model"

			_, err := agent.RunLoop(context.Background(), []agent.AgentMessage{
				agent.NewUserMessage("run echo"),
			}, &agent.Context{}, cfg, agent.LoopDeps{
				Model: chatModel,
				ResolveModel: func(_ context.Context, name string) (llms.Model, error) {
					switch name {
					case "tool-model":
						return toolModel, nil
					default:
						return chatModel, nil
					}
				},
				Registry: reg,
			}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantChatCalls, chatModel.Calls())
			assert.Equal(t, tt.wantToolCalls, toolModel.Calls())
		})
	}
}
