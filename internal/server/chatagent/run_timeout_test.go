package chatagent

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestRunTimeout(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ChatAgentConfig
		want time.Duration
	}{
		{name: "default when unset", cfg: config.ChatAgentConfig{}, want: DefaultRunTimeout},
		{name: "default when zero", cfg: config.ChatAgentConfig{RunTimeout: 0}, want: DefaultRunTimeout},
		{name: "custom run timeout", cfg: config.ChatAgentConfig{RunTimeout: 5 * time.Minute}, want: 5 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LockAppConfigForTest(t)

			orig := config.App.ChatAgent
			config.App.ChatAgent = tt.cfg
			t.Cleanup(func() { config.App.ChatAgent = orig })

			assert.Equal(t, tt.want, RunTimeout())
		})
	}
}
