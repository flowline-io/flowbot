package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
)

func TestAgentModelName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agentName string
		chatAgent config.ChatAgentConfig
		wantModel string
	}{
		{
			name:      "chat agent with chat_model",
			agentName: "chat",
			chatAgent: config.ChatAgentConfig{ChatModel: "gpt-5.5-instant"},
			wantModel: "gpt-5.5-instant",
		},
		{
			name:      "unknown agent name",
			agentName: "nonexistent",
			wantModel: "",
		},
		{
			name:      "chat agent without chat_model",
			agentName: "chat",
			chatAgent: config.ChatAgentConfig{},
			wantModel: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chatAgent := config.App.ChatAgent
			t.Cleanup(func() {
				config.App.ChatAgent = chatAgent
			})
			config.App.ChatAgent = tt.chatAgent
			got := llm.AgentModelName(tt.agentName)
			assert.Equal(t, tt.wantModel, got)
		})
	}
}

func TestAgentEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agentName string
		chatAgent config.ChatAgentConfig
		want      bool
	}{
		{
			name:      "chat enabled via chat_model",
			agentName: "chat",
			chatAgent: config.ChatAgentConfig{ChatModel: "gpt-5.5-instant"},
			want:      true,
		},
		{
			name:      "chat disabled without chat_model",
			agentName: "chat",
			chatAgent: config.ChatAgentConfig{},
			want:      false,
		},
		{
			name:      "unknown agent name",
			agentName: "nonexistent",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chatAgent := config.App.ChatAgent
			t.Cleanup(func() {
				config.App.ChatAgent = chatAgent
			})
			config.App.ChatAgent = tt.chatAgent
			got := llm.AgentEnabled(tt.agentName)
			assert.Equal(t, tt.want, got)
		})
	}
}
