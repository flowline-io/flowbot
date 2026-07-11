package llm

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestLLMHTTPTimeout(t *testing.T) {
	prev := config.App.ChatAgent.RunTimeout
	t.Cleanup(func() { config.App.ChatAgent.RunTimeout = prev })

	tests := []struct {
		name       string
		runTimeout time.Duration
		want       time.Duration
	}{
		{
			name:       "default when unset",
			runTimeout: 0,
			want:       defaultLLMHTTPTimeout,
		},
		{
			name:       "uses configured run timeout",
			runTimeout: 5 * time.Minute,
			want:       5 * time.Minute,
		},
		{
			name:       "custom short timeout",
			runTimeout: 30 * time.Second,
			want:       30 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.RunTimeout = tt.runTimeout
			assert.Equal(t, tt.want, llmHTTPTimeout())
		})
	}
}
