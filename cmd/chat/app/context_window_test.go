package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestResolveContextWindow(t *testing.T) {
	tests := []struct {
		name      string
		chatModel string
		toolModel string
		want      int
	}{
		{name: "flash chat model", chatModel: "deepseek-v4-flash", want: 1_048_576},
		{name: "dual model uses max", chatModel: "deepseek-v4-flash", toolModel: "deepseek-v4-pro", want: 1_048_576},
		{name: "unknown model fallback", chatModel: "fake-model", want: model.DefaultContextWindow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveContextWindow(tt.chatModel, tt.toolModel))
		})
	}
}

func TestResolveContextWindowFromInfo(t *testing.T) {
	tests := []struct {
		name string
		info *client.ChatAgentInfo
		want int
	}{
		{
			name: "from agent info",
			info: &client.ChatAgentInfo{ChatModel: "deepseek-v4-flash", ToolModel: "deepseek-v4-pro"},
			want: 1_048_576,
		},
		{name: "nil info fallback", info: nil, want: model.DefaultContextWindow},
		{name: "codex model", info: &client.ChatAgentInfo{ChatModel: "gpt-5.3-codex"}, want: 400_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveContextWindowFromInfo(tt.info))
		})
	}
}
