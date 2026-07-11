package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeTokenUsageSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "agent canonical", input: "agent", want: TokenUsageSourceAgent},
		{name: "legacy chat agent", input: "chat_agent", want: TokenUsageSourceAgent},
		{name: "pipeline", input: "pipeline", want: TokenUsageSourcePipeline},
		{name: "pipeline uppercase", input: "Pipeline", want: TokenUsageSourcePipeline},
		{name: "scheduled task", input: "scheduled_task", want: TokenUsageSourceScheduledTask},
		{name: "subagent", input: "subagent", want: TokenUsageSourceSubagent},
		{name: "empty defaults agent", input: "", want: TokenUsageSourceAgent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, NormalizeTokenUsageSource(tt.input))
		})
	}
}

func TestTokenUsageSourceLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "agent label", input: "agent", want: "Agent"},
		{name: "pipeline label", input: "pipeline", want: "Pipeline"},
		{name: "scheduled label", input: "scheduled_task", want: "Scheduled Task"},
		{name: "subagent label", input: "subagent", want: "Subagent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TokenUsageSourceLabel(tt.input))
		})
	}
}
