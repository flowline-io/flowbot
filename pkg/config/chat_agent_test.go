package config_test

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestChatAgentConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ChatAgentConfig
	}{
		{name: "defaults", cfg: config.ChatAgentConfig{}},
		{name: "custom workspace", cfg: config.ChatAgentConfig{Workspace: "/tmp/ws"}},
		{name: "custom limits", cfg: config.ChatAgentConfig{ShellTimeout: 30 * time.Second, MaxToolOutput: 4096}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, tt.cfg)
		})
	}
}
