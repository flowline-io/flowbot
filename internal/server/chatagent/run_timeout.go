package chatagent

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
)

// DefaultRunTimeout is the maximum duration for one assistant turn when not configured.
const DefaultRunTimeout = 10 * time.Minute

// harnessDrainTimeout bounds how long a cancelled run waits for the pooled harness to idle.
const harnessDrainTimeout = 30 * time.Second

// RunTimeout returns the configured per-turn timeout for the chat assistant.
func RunTimeout() time.Duration {
	timeout := config.App.ChatAgent.RunTimeout
	if timeout <= 0 {
		return DefaultRunTimeout
	}
	return timeout
}
