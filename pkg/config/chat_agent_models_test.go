package config_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveChatAgentModels(t *testing.T) {
	openAIModels := []config.Model{
		{
			Provider:   "openai",
			ModelNames: []string{"gpt-4o-mini", "gpt-4o"},
		},
	}
	mixedModels := []config.Model{
		{Provider: "openai", ModelNames: []string{"gpt-4o-mini"}},
		{Provider: "anthropic", ModelNames: []string{"claude-opus-4"}},
	}

	tests := []struct {
		name      string
		agents    []config.Agent
		chatAgent config.ChatAgentConfig
		models    []config.Model
		wantChat  string
		wantTool  string
		wantDual  bool
		wantErr   string
	}{
		{
			name:     "agent model only",
			agents:   []config.Agent{{Name: "chat", Enabled: true, Model: "gpt-4o-mini"}},
			models:   openAIModels,
			wantChat: "gpt-4o-mini",
			wantDual: false,
		},
		{
			name: "dual models configured",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4o-mini"},
			},
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "gpt-4o",
			},
			models:   openAIModels,
			wantChat: "gpt-4o-mini",
			wantTool: "gpt-4o",
			wantDual: true,
		},
		{
			name: "empty tool model",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4o-mini"},
			},
			chatAgent: config.ChatAgentConfig{ToolModel: ""},
			models:    openAIModels,
			wantChat:  "gpt-4o-mini",
			wantDual:  false,
		},
		{
			name: "same chat and tool model",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4o-mini"},
			},
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "gpt-4o-mini",
			},
			models:   openAIModels,
			wantChat: "gpt-4o-mini",
			wantTool: "gpt-4o-mini",
			wantDual: false,
		},
		{
			name: "chat model override",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4o"},
			},
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "gpt-4o",
			},
			models:   openAIModels,
			wantChat: "gpt-4o-mini",
			wantTool: "gpt-4o",
			wantDual: true,
		},
		{
			name: "unregistered tool model",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4o-mini"},
			},
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "missing-model",
			},
			models:  openAIModels,
			wantErr: `tool model "missing-model" is not registered`,
		},
		{
			name: "cross provider rejected",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4o-mini"},
			},
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "claude-opus-4",
			},
			models:  mixedModels,
			wantErr: "must use the same provider",
		},
		{
			name: "unregistered chat model in single mode",
			agents: []config.Agent{
				{Name: "chat", Enabled: true, Model: "missing-chat"},
			},
			models:  openAIModels,
			wantErr: `chat model "missing-chat" is not registered`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config.App.Agents = tt.agents
			config.App.ChatAgent = tt.chatAgent
			config.App.Models = tt.models

			chat, tool, dual, err := config.ResolveChatAgentModels()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantChat, chat)
			assert.Equal(t, tt.wantTool, tool)
			assert.Equal(t, tt.wantDual, dual)
		})
	}
}

func TestModelRegisteredAndProviderFor(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		wantProvider string
		wantReg      bool
	}{
		{name: "registered model", modelName: "gpt-4o", wantProvider: "openai", wantReg: true},
		{name: "unknown model", modelName: "missing", wantProvider: "", wantReg: false},
		{name: "second provider model", modelName: "claude", wantProvider: "anthropic", wantReg: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config.App.Models = []config.Model{
				{Provider: "openai", ModelNames: []string{"gpt-4o"}},
				{Provider: "anthropic", ModelNames: []string{"claude"}},
			}
			assert.Equal(t, tt.wantReg, config.ModelRegistered(tt.modelName))
			assert.Equal(t, tt.wantProvider, config.ModelProviderFor(tt.modelName))
		})
	}
}

func TestChatAgentChatModelAndEnabled(t *testing.T) {
	tests := []struct {
		name      string
		agents    []config.Agent
		chatAgent config.ChatAgentConfig
		wantModel string
		wantEn    bool
	}{
		{
			name:      "chat_model override without agents model",
			agents:    []config.Agent{{Name: "chat", Enabled: true, Model: ""}},
			chatAgent: config.ChatAgentConfig{ChatModel: "gpt-4o-mini"},
			wantModel: "gpt-4o-mini",
			wantEn:    true,
		},
		{
			name:      "agents model when chat_model empty",
			agents:    []config.Agent{{Name: "chat", Enabled: true, Model: "gpt-4o"}},
			chatAgent: config.ChatAgentConfig{},
			wantModel: "gpt-4o",
			wantEn:    true,
		},
		{
			name:      "disabled chat agent",
			agents:    []config.Agent{{Name: "chat", Enabled: false, Model: "gpt-4o"}},
			chatAgent: config.ChatAgentConfig{ChatModel: "gpt-4o-mini"},
			wantModel: "gpt-4o-mini",
			wantEn:    false,
		},
		{
			name:      "enabled without any model",
			agents:    []config.Agent{{Name: "chat", Enabled: true, Model: ""}},
			chatAgent: config.ChatAgentConfig{},
			wantModel: "",
			wantEn:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config.App.Agents = tt.agents
			config.App.ChatAgent = tt.chatAgent
			assert.Equal(t, tt.wantModel, config.ChatAgentChatModel())
			assert.Equal(t, tt.wantEn, config.ChatAgentEnabled())
		})
	}
}
