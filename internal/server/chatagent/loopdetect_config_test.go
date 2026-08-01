package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestLoopDetectConfigFromApp(t *testing.T) {
	LockAppConfigForTest(t)

	t.Run("defaults when unset", func(t *testing.T) {
		config.App.ChatAgent.LoopDetection = config.ChatAgentLoopDetectionConfig{}
		got := loopDetectConfigFromApp()
		assert.True(t, got.Enabled)
		assert.Equal(t, 30, got.Window)
		assert.Equal(t, 10, got.GenericCritical)
	})

	t.Run("explicit disable", func(t *testing.T) {
		enabled := false
		config.App.ChatAgent.LoopDetection = config.ChatAgentLoopDetectionConfig{Enabled: &enabled}
		got := loopDetectConfigFromApp()
		assert.False(t, got.Enabled)
	})

	t.Run("custom thresholds", func(t *testing.T) {
		enabled := true
		config.App.ChatAgent.LoopDetection = config.ChatAgentLoopDetectionConfig{
			Enabled:                 &enabled,
			Window:                  12,
			GenericCritical:         7,
			GlobalCircuitBreaker:    15,
			PostCompactionWatch:     4,
			PostCompactionIdentical: 2,
		}
		got := loopDetectConfigFromApp()
		assert.True(t, got.Enabled)
		assert.Equal(t, 12, got.Window)
		assert.Equal(t, 7, got.GenericCritical)
		assert.Equal(t, 15, got.GlobalCircuitBreaker)
		assert.Equal(t, 4, got.PostCompactionWatch)
		assert.Equal(t, 2, got.PostCompactionIdentical)
	})
}
