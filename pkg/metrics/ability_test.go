package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewAbilityCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewAbilityCollector(nil)
		assert.NotNil(t, c)
		c.IncInvokeTotal("bookmark", "list", "ok")
		c.ObserveInvokeDuration("bookmark", "list", 0.5)
		c.IncInvokeError("bookmark", "list", "timeout")
	})
}

func TestAbilityCollector_CounterMetrics(t *testing.T) {
	invokeTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ability_invoke_total",
			Help: "Invocations by capability, operation, and status",
		},
		[]string{"capability", "operation", "status"},
	)
	c := &AbilityCollector{invokeTotal: invokeTotal}

	c.IncInvokeTotal("bookmark", "list", "ok")
	c.IncInvokeTotal("bookmark", "list", "ok")
	c.IncInvokeTotal("bookmark", "create", "ok")
	c.IncInvokeTotal("kanban", "list", "ok")

	expected := `
# HELP ability_invoke_total Invocations by capability, operation, and status
# TYPE ability_invoke_total counter
ability_invoke_total{capability="bookmark",operation="list",status="ok"} 2
ability_invoke_total{capability="bookmark",operation="create",status="ok"} 1
ability_invoke_total{capability="kanban",operation="list",status="ok"} 1
`
	err := testutil.CollectAndCompare(c.invokeTotal, strings.NewReader(expected), "ability_invoke_total")
	assert.NoError(t, err)
}

func TestAbilityCollector_ErrorMetrics(t *testing.T) {
	invokeErrorTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ability_invoke_error_total",
			Help: "Invocation errors by capability, operation, and error code",
		},
		[]string{"capability", "operation", "error_code"},
	)
	c := &AbilityCollector{invokeErrorTotal: invokeErrorTotal}

	c.IncInvokeError("bookmark", "list", "timeout")
	c.IncInvokeError("bookmark", "list", "timeout")
	c.IncInvokeError("bookmark", "list", "rate_limited")

	expected := `
# HELP ability_invoke_error_total Invocation errors by capability, operation, and error code
# TYPE ability_invoke_error_total counter
ability_invoke_error_total{capability="bookmark",error_code="rate_limited",operation="list"} 1
ability_invoke_error_total{capability="bookmark",error_code="timeout",operation="list"} 2
`
	err := testutil.CollectAndCompare(c.invokeErrorTotal, strings.NewReader(expected), "ability_invoke_error_total")
	assert.NoError(t, err)
}

func TestAbilityCollector_IncEventDropped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setup         func() (*AbilityCollector, *prometheus.CounterVec)
		capability    string
		operation     string
		reason        string
		callCount     int
		wantNoPanic   bool
		expectedText  string
		metricName    string
	}{
		{
			name:        "no-op collector does not panic on nil stats",
			setup: func() (*AbilityCollector, *prometheus.CounterVec) {
				return NewAbilityCollector(nil), nil
			},
			capability:  "bookmark",
			operation:   "list",
			reason:      "pool_full",
			callCount:   1,
			wantNoPanic: true,
		},
		{
			name: "single increment registers counter",
			setup: func() (*AbilityCollector, *prometheus.CounterVec) {
				cv := prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "ability_event_dropped_total",
						Help: "Events dropped due to pool overflow or shutdown",
					},
					[]string{"capability", "operation", "reason"},
				)
				return &AbilityCollector{eventDroppedTotal: cv}, cv
			},
			capability: "bookmark",
			operation:  "list",
			reason:     "pool_full",
			callCount:  1,
			expectedText: `
# HELP ability_event_dropped_total Events dropped due to pool overflow or shutdown
# TYPE ability_event_dropped_total counter
ability_event_dropped_total{capability="bookmark",operation="list",reason="pool_full"} 1
`,
			metricName: "ability_event_dropped_total",
		},
		{
			name: "multiple increments with different labels",
			setup: func() (*AbilityCollector, *prometheus.CounterVec) {
				cv := prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "ability_event_dropped_total",
						Help: "Events dropped due to pool overflow or shutdown",
					},
					[]string{"capability", "operation", "reason"},
				)
				return &AbilityCollector{eventDroppedTotal: cv}, cv
			},
			capability: "bookmark",
			operation:  "list",
			reason:     "pool_full",
			callCount:  3,
			expectedText: `
# HELP ability_event_dropped_total Events dropped due to pool overflow or shutdown
# TYPE ability_event_dropped_total counter
ability_event_dropped_total{capability="bookmark",operation="list",reason="pool_full"} 3
`,
			metricName: "ability_event_dropped_total",
		},
		{
			name: "different reasons increment different series",
			setup: func() (*AbilityCollector, *prometheus.CounterVec) {
				cv := prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "ability_event_dropped_total",
						Help: "Events dropped due to pool overflow or shutdown",
					},
					[]string{"capability", "operation", "reason"},
				)
				return &AbilityCollector{eventDroppedTotal: cv}, cv
			},
			capability: "kanban",
			operation:  "create",
			reason:     "shutdown",
			callCount:  2,
			expectedText: `
# HELP ability_event_dropped_total Events dropped due to pool overflow or shutdown
# TYPE ability_event_dropped_total counter
ability_event_dropped_total{capability="kanban",operation="create",reason="shutdown"} 2
`,
			metricName: "ability_event_dropped_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, cv := tt.setup()
			if tt.wantNoPanic {
				assert.NotPanics(t, func() {
					for i := 0; i < tt.callCount; i++ {
						c.IncEventDropped(tt.capability, tt.operation, tt.reason)
					}
				})
				return
			}
			for i := 0; i < tt.callCount; i++ {
				c.IncEventDropped(tt.capability, tt.operation, tt.reason)
			}
			err := testutil.CollectAndCompare(cv, strings.NewReader(tt.expectedText), tt.metricName)
			assert.NoError(t, err)
		})
	}
}

func TestAbilityCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewAbilityCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncInvokeTotal", fn: func() { c.IncInvokeTotal("c", "o", "ok") }},
		{name: "ObserveInvokeDuration", fn: func() { c.ObserveInvokeDuration("c", "o", 1.0) }},
		{name: "IncInvokeError", fn: func() { c.IncInvokeError("c", "o", "err") }},
		{name: "IncEventDropped", fn: func() { c.IncEventDropped("c", "o", "pool_full") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
