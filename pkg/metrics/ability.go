// Package metrics provides Prometheus metrics collection for abilities.
package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// AbilityCollector holds typed metrics for ability invocations.
// When initialized with a nil stats, all methods are no-op.
type AbilityCollector struct {
	invokeTotal       *prometheus.CounterVec
	invokeDuration    *prometheus.HistogramVec
	invokeErrorTotal  *prometheus.CounterVec
	eventDroppedTotal *prometheus.CounterVec
}

// NewAbilityCollector creates an AbilityCollector backed by stats.
// Returns a no-op collector when stats is nil or if registration fails.
func NewAbilityCollector(st *stats.Stats) *AbilityCollector {
	if st == nil {
		return &AbilityCollector{}
	}
	var err error
	c := &AbilityCollector{}
	c.invokeTotal, err = st.RegisterCounterVec("ability_invoke_total", "Invocations by capability, operation, and status", "capability", "operation", "status")
	if err != nil {
		log.Printf("[metrics] ability: failed to register counter vec: %v", err)
		return &AbilityCollector{}
	}
	c.invokeDuration, err = st.RegisterHistogramVec("ability_invoke_duration_seconds", "Invocation duration distribution", "capability", "operation")
	if err != nil {
		log.Printf("[metrics] ability: failed to register histogram vec: %v", err)
		return &AbilityCollector{}
	}
	c.invokeErrorTotal, err = st.RegisterCounterVec("ability_invoke_error_total", "Invocation errors by capability, operation, and error code", "capability", "operation", "error_code")
	if err != nil {
		log.Printf("[metrics] ability: failed to register counter vec: %v", err)
		return &AbilityCollector{}
	}
	c.eventDroppedTotal, err = st.RegisterCounterVec("ability_event_dropped_total", "Events dropped due to pool overflow or shutdown", "capability", "operation", "reason")
	if err != nil {
		log.Printf("[metrics] ability: failed to register counter vec: %v", err)
		return &AbilityCollector{}
	}
	return c
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

// IncEventDropped increments the event dropped counter.
func (c *AbilityCollector) IncEventDropped(capability, operation, reason string) {
	if c.eventDroppedTotal == nil {
		return
	}
	defer recoverLog("ability_event_dropped_total")
	c.eventDroppedTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(reason)).Inc()
}

// IncInvokeError increments the error counter for the given capability, operation, and error code.
func (c *AbilityCollector) IncInvokeError(capability, operation, errorCode string) {
	if c.invokeErrorTotal == nil {
		return
	}
	defer recoverLog("ability_invoke_error_total")
	c.invokeErrorTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(errorCode)).Inc()
}
