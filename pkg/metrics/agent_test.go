package metrics_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/stretchr/testify/assert"
)

func TestAgentCollectorNoop(t *testing.T) {
	t.Parallel()
	c := metrics.NewAgentCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncRunTotal", fn: func() { c.IncRunTotal("ok") }},
		{name: "ObserveTurnDuration", fn: func() { c.ObserveTurnDuration("ok", 0.1) }},
		{name: "IncLLMRequest", fn: func() { c.IncLLMRequest("m", "ok") }},
		{name: "IncLLMRetry", fn: func() { c.IncLLMRetry("m") }},
		{name: "ObserveLLMDuration", fn: func() { c.ObserveLLMDuration("m", 0.2) }},
		{name: "IncToolTotal", fn: func() { c.IncToolTotal("echo", "ok") }},
		{name: "ObserveToolDuration", fn: func() { c.ObserveToolDuration("echo", 0.3) }},
		{name: "IncCompact", fn: func() { c.IncCompact("ok") }},
		{name: "IncOverflowRetry", fn: func() { c.IncOverflowRetry("1") }},
		{name: "IncDoomLoop", fn: func() { c.IncDoomLoop("run_terminal") }},
		{name: "IncSensorLint", fn: func() { c.IncSensorLint("ok") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, tt.fn)
		})
	}
}

func TestDefaultAgentCollector(t *testing.T) {
	t.Parallel()
	metrics.SetDefaultAgentCollector(nil)
	assert.NotNil(t, metrics.Agent())
	c := metrics.NewAgentCollector(nil)
	metrics.SetDefaultAgentCollector(c)
	assert.Equal(t, c, metrics.DefaultAgentCollector())
	metrics.SetDefaultAgentCollector(nil)
}
