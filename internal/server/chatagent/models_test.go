package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentLoopConfig(t *testing.T) {
	openAIModels := []config.Model{
		{Provider: "openai", ModelNames: []string{"gpt-4o-mini", "gpt-4o"}},
	}

	tests := []struct {
		name          string
		chatAgent     config.ChatAgentConfig
		models        []config.Model
		wantDual      bool
		wantChatModel string
		wantToolModel string
		wantChatCfg   string
		wantToolCfg   string
		wantErr       string
	}{
		{
			name: "single model from chat_model",
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
			},
			models:        openAIModels,
			wantDual:      false,
			wantChatModel: "gpt-4o-mini",
			wantChatCfg:   "",
			wantToolCfg:   "",
		},
		{
			name: "dual model config without harness router",
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "gpt-4o",
			},
			models:        openAIModels,
			wantDual:      true,
			wantChatModel: "gpt-4o-mini",
			wantToolModel: "gpt-4o",
			wantChatCfg:   "gpt-4o-mini",
			wantToolCfg:   "gpt-4o",
		},
		{
			name: "invalid dual model config",
			chatAgent: config.ChatAgentConfig{
				ChatModel: "gpt-4o-mini",
				ToolModel: "missing",
			},
			models:  openAIModels,
			wantErr: "not registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config.App.ChatAgent = tt.chatAgent
			config.App.Models = tt.models

			cfg, chatModel, toolModel, dual, err := agentLoopConfig()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDual, dual)
			assert.Equal(t, tt.wantChatModel, chatModel)
			assert.Equal(t, tt.wantToolModel, toolModel)
			assert.Equal(t, tt.wantChatCfg, cfg.ChatModel)
			assert.Equal(t, tt.wantToolCfg, cfg.ToolModel)
			assert.Equal(t, tt.wantChatModel, cfg.ModelName)
		})
	}
}
