package llm_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
)

func TestValidThinkingLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  bool
	}{
		{name: "default", level: "default", want: true},
		{name: "off", level: "off", want: true},
		{name: "high uppercase accepted", level: "HIGH", want: true},
		{name: "empty is valid", level: "", want: true},
		{name: "unknown is invalid", level: "turbo", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, llm.ValidThinkingLevel(tt.level))
		})
	}
}

func TestReasoningCallOptionsThinkingLevel(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		level     string
		wantLen   int
	}{
		{name: "off disables anthropic reasoning", modelName: "claude-sonnet-4.6", level: "off", wantLen: 0},
		{name: "default keeps anthropic reasoning", modelName: "claude-sonnet-4.6", level: "default", wantLen: 4},
		{name: "high enables deepseek reasoning", modelName: "deepseek-v4-flash", level: "high", wantLen: 2},
		{name: "off disables deepseek reasoning", modelName: "deepseek-v4-flash", level: "off", wantLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := llm.ReasoningCallOptions(tt.modelName, 4096, tt.level)
			assert.Len(t, opts, tt.wantLen)
		})
	}
}

func TestThinkingLevelContext(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  string
	}{
		{name: "default passthrough", level: "default", want: "default"},
		{name: "low passthrough", level: "low", want: "low"},
		{name: "empty normalises to default", level: "", want: "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := llm.WithThinkingLevel(t.Context(), tt.level)
			assert.Equal(t, tt.want, llm.ThinkingLevelFromContext(ctx))
		})
	}
}
