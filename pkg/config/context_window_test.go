package config_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestCompactionConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name           string
		cfg            config.CompactionConfig
		wantReserve    int
		wantKeepRecent int
	}{
		{name: "zero values", cfg: config.CompactionConfig{}, wantReserve: 16384, wantKeepRecent: 20000},
		{name: "custom reserve", cfg: config.CompactionConfig{ReserveTokens: 8192}, wantReserve: 8192, wantKeepRecent: 20000},
		{name: "custom keep recent", cfg: config.CompactionConfig{KeepRecentTokens: 10000}, wantReserve: 16384, wantKeepRecent: 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cfg.WithDefaults()
			assert.Equal(t, tt.wantReserve, got.ReserveTokens)
			assert.Equal(t, tt.wantKeepRecent, got.KeepRecentTokens)
		})
	}
}

func TestContextWindowForModel(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      int
	}{
		{name: "catalog model", modelName: "deepseek-v4-pro", want: 1_048_576},
		{name: "unknown model fallback", modelName: "unknown", want: model.DefaultContextWindow},
		{name: "empty name fallback", modelName: "", want: model.DefaultContextWindow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.ContextWindowForModels(nil, tt.modelName))
			assert.Equal(t, tt.want, config.ContextWindowForModel(tt.modelName))
		})
	}
}

func TestMaxContextWindow(t *testing.T) {
	tests := []struct {
		name       string
		modelNames []string
		want       int
	}{
		{
			name:       "returns largest catalog window",
			modelNames: []string{"fake-model", "deepseek-v4-pro"},
			want:       1_048_576,
		},
		{
			name:       "falls back when names empty",
			modelNames: nil,
			want:       model.DefaultContextWindow,
		},
		{
			name:       "uses default for unknown models",
			modelNames: []string{"missing-a", "missing-b"},
			want:       model.DefaultContextWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.MaxContextWindow(tt.modelNames...))
		})
	}
}

func TestChatAgentContextWindow(t *testing.T) {
	prev := config.App
	t.Cleanup(func() { config.App = prev })

	tests := []struct {
		name      string
		chatModel string
		toolModel string
		want      int
	}{
		{name: "single chat model", chatModel: "deepseek-v4-flash", want: 1_048_576},
		{name: "dual model uses max", chatModel: "gpt-5.3-codex", toolModel: "deepseek-v4-pro", want: 1_048_576},
		{name: "unknown chat model fallback", chatModel: "missing-model", want: model.DefaultContextWindow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config.App.ChatAgent = config.ChatAgentConfig{
				ChatModel: tt.chatModel,
				ToolModel: tt.toolModel,
			}
			assert.Equal(t, tt.want, config.ChatAgentContextWindow())
		})
	}
}
