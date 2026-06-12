package config_test

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestChatAgentConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.ChatAgentConfig
		wantSteps int
	}{
		{name: "defaults", cfg: config.ChatAgentConfig{}, wantSteps: 0},
		{name: "custom workspace", cfg: config.ChatAgentConfig{Workspace: "/tmp/ws"}, wantSteps: 0},
		{name: "custom limits", cfg: config.ChatAgentConfig{
			RunTimeout: 10 * time.Minute, ShellTimeout: 30 * time.Second, MaxToolOutput: 4096, MaxSteps: 15,
			Compaction: config.CompactionConfig{Enabled: true, ReserveTokens: 8192},
		}, wantSteps: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantSteps, tt.cfg.MaxSteps)
			if tt.cfg.Workspace != "" {
				assert.NotEmpty(t, tt.cfg.Workspace)
			}
		})
	}
}
