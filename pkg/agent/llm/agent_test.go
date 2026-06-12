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
		wantModel string
	}{
		{
			name:      "agent exists and enabled",
			agentName: "agent_active",
			wantModel: "gpt-5.5-instant",
		},
		{
			name:      "agent does not exist",
			agentName: "nonexistent",
			wantModel: "",
		},
		{
			name:      "agent disabled",
			agentName: "agent_disabled",
			wantModel: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
		want      bool
	}{
		{
			name:      "agent active with model",
			agentName: "agent_active",
			want:      true,
		},
		{
			name:      "agent disabled",
			agentName: "agent_disabled",
			want:      false,
		},
		{
			name:      "agent enabled but no model",
			agentName: "agent_nomodel",
			want:      false,
		},
		{
			name:      "chat enabled via chat_agent chat_model only",
			agentName: "chat",
			want:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.agentName == "chat" {
				agents := config.App.Agents
				chatAgent := config.App.ChatAgent
				t.Cleanup(func() {
					config.App.Agents = agents
					config.App.ChatAgent = chatAgent
				})
				config.App.Agents = []config.Agent{
					{Name: "chat", Enabled: true, Model: ""},
				}
				config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-5.5-instant"}
			} else {
				t.Parallel()
			}
			got := llm.AgentEnabled(tt.agentName)
			assert.Equal(t, tt.want, got)
		})
	}
}
