package app

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestRenderContextUsage(t *testing.T) {
	styles := NewStyles()
	info := &client.ChatContextUsage{
		Model:         "deepseek-v4-pro",
		ContextWindow: 1_000_000,
		TotalTokens:   23200,
		TotalPercent:  2.32,
		Categories: []client.ChatContextCategory{
			{ID: "system_prompt", Label: "System prompt", Tokens: 5800, Percent: 0.6},
			{ID: "system_tools", Label: "System tools", Tokens: 16200, Percent: 1.6},
			{ID: "messages", Label: "Messages", Tokens: 1200, Percent: 0.12},
			{ID: "free_space", Label: "Free space", Tokens: 943800, Percent: 94.4},
			{ID: "autocompact_buffer", Label: "Autocompact buffer", Tokens: 33000, Percent: 3.3},
		},
		Skills: []client.ChatContextSkill{
			{Name: "run", Tokens: 120},
			{Name: "loop", Tokens: 100},
		},
	}

	tests := []struct {
		name    string
		info    *client.ChatContextUsage
		wantSub []string
	}{
		{
			name: "renders title and model",
			info: info,
			wantSub: []string{
				"Context Usage",
				"deepseek-v4-pro[1m]",
				"23.2k/1m tokens",
				"Estimated usage by category",
				"System prompt:",
				"Autocompact buffer:",
				"Skills · /skills",
				"├ run:",
			},
		},
		{
			name:    "nil info",
			info:    nil,
			wantSub: []string{"Context usage unavailable"},
		},
		{
			name: "empty skills omits tree",
			info: &client.ChatContextUsage{
				Model:         "test",
				ContextWindow: 128000,
				TotalTokens:   1000,
				TotalPercent:  0.78,
				Categories: []client.ChatContextCategory{
					{ID: "messages", Label: "Messages", Tokens: 1000, Percent: 0.78},
				},
			},
			wantSub: []string{"Context Usage", "test[128k]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderContextUsage(tt.info, styles)
			for _, sub := range tt.wantSub {
				assert.Contains(t, got, sub)
			}
			if tt.info != nil && len(tt.info.Skills) == 0 {
				assert.NotContains(t, got, "Built-in")
			}
		})
	}
}

func TestRenderContextUsageBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		wantSub string
	}{
		{name: "zero usage", percent: 0, wantSub: string(blockEmpty)},
		{name: "partial usage", percent: 2.32, wantSub: string(blockPartial)},
		{name: "high usage", percent: 50, wantSub: string(blockFull)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderContextUsageBar(tt.percent, contextBarWidth)
			assert.Contains(t, got, tt.wantSub)
			assert.Equal(t, contextBarWidth-1, strings.Count(got, " "))
		})
	}
}

func TestFormatContextTokenCount(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{name: "millions", tokens: 1_000_000, want: "1.0m"},
		{name: "thousands", tokens: 23200, want: "23.2k"},
		{name: "small", tokens: 154, want: "154"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatContextTokenCount(tt.tokens))
		})
	}
}

func TestFormatSkillTokenEstimate(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{name: "under threshold", tokens: 10, want: "< 20 tokens"},
		{name: "normal", tokens: 250, want: "~250 tokens"},
		{name: "large", tokens: 1200, want: "~1.2k tokens"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSkillTokenEstimate(tt.tokens))
		})
	}
}
