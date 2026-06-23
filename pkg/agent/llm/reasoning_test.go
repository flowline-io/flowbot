package llm_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestSupportsReasoningStream(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      bool
	}{
		{name: "gpt-4o is not reasoning", modelName: "gpt-4o", want: false},
		{name: "gpt-5 is reasoning", modelName: "gpt-5.3-codex", want: true},
		{name: "claude-sonnet-4 is reasoning", modelName: "claude-sonnet-4", want: true},
		{name: "deepseek-v4-flash is reasoning", modelName: "deepseek-v4-flash", want: true},
		{name: "deepseek-v4-pro is reasoning", modelName: "deepseek-v4-pro", want: true},
		{name: "deepseek-chat is not reasoning", modelName: "deepseek-chat", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, llm.SupportsReasoningStream(tt.modelName))
		})
	}
}

func TestReasoningCallOptions(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		maxTokens int
		wantLen   int
		wantMode  llms.ThinkingMode
	}{
		{
			name:      "non reasoning model returns nil",
			modelName: "gpt-4o",
			maxTokens: 4096,
			wantLen:   0,
		},
		{
			name:      "openai reasoning model enables stream thinking",
			modelName: "gpt-5.3-codex",
			maxTokens: 4096,
			wantLen:   2,
		},
		{
			name:      "deepseek v4 flash enables stream thinking",
			modelName: "deepseek-v4-flash",
			maxTokens: 4096,
			wantLen:   2,
		},
		{
			name:      "anthropic reasoning model adds thinking mode",
			modelName: "claude-sonnet-4",
			maxTokens: 8192,
			wantLen:   4,
			wantMode:  llms.ThinkingModeAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := llm.ReasoningCallOptions(tt.modelName, tt.maxTokens)
			assert.Len(t, opts, tt.wantLen)
			if tt.wantMode == "" {
				return
			}

			callOpts := llms.CallOptions{}
			for _, opt := range opts {
				opt(&callOpts)
			}
			cfg := llms.GetThinkingConfig(&callOpts)
			if assert.NotNil(t, cfg) {
				assert.Equal(t, tt.wantMode, cfg.Mode)
			}
		})
	}
}
