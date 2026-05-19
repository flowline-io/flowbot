package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// AbilityCollector holds typed metrics for ability invocations.
// When initialized with a nil stats, all methods are no-op.
type AbilityCollector struct {
	invokeTotal      *prometheus.CounterVec
	invokeDuration   *prometheus.HistogramVec
	invokeErrorTotal *prometheus.CounterVec
}

// NewAbilityCollector creates an AbilityCollector backed by stats.
// Returns a no-op collector when stats is nil.
func NewAbilityCollector(st *stats.Stats) *AbilityCollector {
	if st == nil {
		return &AbilityCollector{}
	}
	return &AbilityCollector{
		invokeTotal:      st.RegisterCounterVec("ability_invoke_total", "Invocations by capability, operation, and status", "capability", "operation", "status"),
		invokeDuration:   st.RegisterHistogramVec("ability_invoke_duration_seconds", "Invocation duration distribution", "capability", "operation"),
		invokeErrorTotal: st.RegisterCounterVec("ability_invoke_error_total", "Invocation errors by capability, operation, and error code", "capability", "operation", "error_code"),
	}
}

// IncInvokeTotal increments the invoke counter for the given capability, operation, and status.
func (c *AbilityCollector) IncInvokeTotal(capability, operation, status string) {
	if c.invokeTotal == nil {
		return
	}
	defer recoverLog("ability_invoke_total")
	c.invokeTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(status)).Inc()
}

// ObserveInvokeDuration records an invocation duration observation.
func (c *AbilityCollector) ObserveInvokeDuration(capability, operation string, seconds float64) {
	if c.invokeDuration == nil {
		return
	}
	defer recoverLog("ability_invoke_duration_seconds")
	c.invokeDuration.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation)).Observe(seconds)
}

// IncInvokeError increments the error counter for the given capability, operation, and error code.
func (c *AbilityCollector) IncInvokeError(capability, operation, errorCode string) {
	if c.invokeErrorTotal == nil {
		return
	}
	defer recoverLog("ability_invoke_error_total")
	c.invokeErrorTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(errorCode)).Inc()
}
