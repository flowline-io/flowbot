package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewCapabilityCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewCapabilityCollector(nil)
		assert.NotNil(t, c)
		c.IncInvokeTotal("karakeep", "list", "ok")
		c.ObserveInvokeDuration("karakeep", "list", 0.5)
		c.IncInvokeError("karakeep", "list", "timeout")
	})
}

func TestCapabilityCollector_CounterMetrics(t *testing.T) {
	invokeTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "capability_invoke_total",
			Help: "Invocations by capability, operation, and status",
		},
		[]string{"capability", "operation", "status"},
	)
	c := &CapabilityCollector{invokeTotal: invokeTotal}

	c.IncInvokeTotal("karakeep", "list", "ok")
	c.IncInvokeTotal("karakeep", "list", "ok")
	c.IncInvokeTotal("karakeep", "create", "ok")
	c.IncInvokeTotal("kanboard", "list", "ok")

	expected := `
# HELP capability_invoke_total Invocations by capability, operation, and status
# TYPE capability_invoke_total counter
capability_invoke_total{capability="karakeep",operation="list",status="ok"} 2
capability_invoke_total{capability="karakeep",operation="create",status="ok"} 1
capability_invoke_total{capability="kanboard",operation="list",status="ok"} 1
`
	err := testutil.CollectAndCompare(c.invokeTotal, strings.NewReader(expected), "capability_invoke_total")
	assert.NoError(t, err)
}

func TestCapabilityCollector_ErrorMetrics(t *testing.T) {
	invokeErrorTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "capability_invoke_error_total",
			Help: "Invocation errors by capability, operation, and error code",
		},
		[]string{"capability", "operation", "error_code"},
	)
	c := &CapabilityCollector{invokeErrorTotal: invokeErrorTotal}

	c.IncInvokeError("karakeep", "list", "timeout")
	c.IncInvokeError("karakeep", "list", "timeout")
	c.IncInvokeError("karakeep", "list", "rate_limited")

	expected := `
# HELP capability_invoke_error_total Invocation errors by capability, operation, and error code
# TYPE capability_invoke_error_total counter
capability_invoke_error_total{capability="karakeep",error_code="rate_limited",operation="list"} 1
capability_invoke_error_total{capability="karakeep",error_code="timeout",operation="list"} 2
`
	err := testutil.CollectAndCompare(c.invokeErrorTotal, strings.NewReader(expected), "capability_invoke_error_total")
	assert.NoError(t, err)
}

func TestCapabilityCollector_IncEventDropped(t *testing.T) {
	eventDroppedTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "capability_event_dropped_total",
			Help: "Events dropped due to pool overflow or shutdown",
		},
		[]string{"capability", "operation", "reason"},
	)
	c := &CapabilityCollector{eventDroppedTotal: eventDroppedTotal}

	c.IncEventDropped("karakeep", "list", "pool_full")
	c.IncEventDropped("karakeep", "list", "pool_full")
	c.IncEventDropped("karakeep", "list", "pool_full")
	c.IncEventDropped("kanboard", "create", "shutdown")
	c.IncEventDropped("kanboard", "create", "shutdown")

	expected := `
# HELP capability_event_dropped_total Events dropped due to pool overflow or shutdown
# TYPE capability_event_dropped_total counter
capability_event_dropped_total{capability="karakeep",operation="list",reason="pool_full"} 3
capability_event_dropped_total{capability="kanboard",operation="create",reason="shutdown"} 2
`
	err := testutil.CollectAndCompare(eventDroppedTotal, strings.NewReader(expected), "capability_event_dropped_total")
	assert.NoError(t, err)
}

func TestCapabilityCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewCapabilityCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncInvokeTotal", fn: func() { c.IncInvokeTotal("c", "o", "ok") }},
		{name: "ObserveInvokeDuration", fn: func() { c.ObserveInvokeDuration("c", "o", 1.0) }},
		{name: "IncInvokeError", fn: func() { c.IncInvokeError("c", "o", "err") }},
		{name: "IncEventDropped", fn: func() { c.IncEventDropped("c", "o", "pool_full") }},
		{name: "IncBulkheadQueued", fn: func() { c.IncBulkheadQueued("c") }},
		{name: "DecBulkheadQueued", fn: func() { c.DecBulkheadQueued("c") }},
		{name: "IncBulkheadActive", fn: func() { c.IncBulkheadActive("c") }},
		{name: "DecBulkheadActive", fn: func() { c.DecBulkheadActive("c") }},
		{name: "IncBulkheadDropped", fn: func() { c.IncBulkheadDropped("c", "timeout") }},
		{name: "ObserveBulkheadWaitDuration", fn: func() { c.ObserveBulkheadWaitDuration("c", 0.5) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
