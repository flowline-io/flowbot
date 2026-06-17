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
		wantAuto       bool
		wantPrune      bool
		wantReserve    int
		wantKeepRecent int
	}{
		{name: "zero values", cfg: config.CompactionConfig{}, wantAuto: true, wantPrune: true, wantReserve: 10000, wantKeepRecent: 20000},
		{name: "legacy enabled and reserve", cfg: config.CompactionConfig{Enabled: new(false), ReserveTokens: 8192}, wantAuto: false, wantPrune: true, wantReserve: 8192, wantKeepRecent: 20000},
		{name: "explicit new fields", cfg: config.CompactionConfig{Auto: new(true), Prune: new(false), Reserved: 12000, KeepRecentTokens: 10000}, wantAuto: true, wantPrune: false, wantReserve: 12000, wantKeepRecent: 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cfg.WithDefaults()
			if assert.NotNil(t, got.Auto) {
				assert.Equal(t, tt.wantAuto, *got.Auto)
			}
			if assert.NotNil(t, got.Prune) {
				assert.Equal(t, tt.wantPrune, *got.Prune)
			}
			assert.Equal(t, tt.wantReserve, got.Reserved)
			assert.Equal(t, tt.wantKeepRecent, got.KeepRecentTokens)
			assert.Equal(t, tt.wantAuto, tt.cfg.AutoEnabled())
			assert.Equal(t, tt.wantPrune, tt.cfg.PruneEnabled())
			assert.Equal(t, tt.wantReserve, tt.cfg.ReservedTokens())
			assert.Equal(t, tt.wantKeepRecent, tt.cfg.KeepRecentBudget())
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
			// Subtests mutate global config.App; do not run in parallel.
			config.App.ChatAgent = config.ChatAgentConfig{
				ChatModel: tt.chatModel,
				ToolModel: tt.toolModel,
			}
			assert.Equal(t, tt.want, config.ChatAgentContextWindow())
		})
	}
}
