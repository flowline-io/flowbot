package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/llm"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := llm.AgentEnabled(tt.agentName)
			assert.Equal(t, tt.want, got)
		})
	}
}
