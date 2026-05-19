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

func TestAbilityCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewAbilityCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncInvokeTotal", fn: func() { c.IncInvokeTotal("c", "o", "ok") }},
		{name: "ObserveInvokeDuration", fn: func() { c.ObserveInvokeDuration("c", "o", 1.0) }},
		{name: "IncInvokeError", fn: func() { c.IncInvokeError("c", "o", "err") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
