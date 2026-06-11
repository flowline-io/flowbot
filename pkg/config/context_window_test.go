package config_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestCompactionConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name              string
		cfg               config.CompactionConfig
		wantReserve       int
		wantKeepRecent    int
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
		models    []config.Model
		modelName string
		want      int
	}{
		{name: "configured model", models: []config.Model{{ContextWindows: map[string]int{"gpt-4o": 64000}}}, modelName: "gpt-4o", want: 64000},
		{name: "unknown model fallback", models: nil, modelName: "unknown", want: 128000},
		{name: "zero window skipped", models: []config.Model{{ContextWindows: map[string]int{"gpt-4o": 0}}}, modelName: "gpt-4o", want: 128000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orig := config.App.Models
			config.App.Models = tt.models
			t.Cleanup(func() { config.App.Models = orig })
			assert.Equal(t, tt.want, config.ContextWindowForModel(tt.modelName))
		})
	}
}
