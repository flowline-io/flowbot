// Package metrics provides Prometheus metrics collection for capabilities.
package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// CapabilityCollector holds typed metrics for capability invocations.
// When initialized with a nil stats, all methods are no-op.
type CapabilityCollector struct {
	invokeTotal          *prometheus.CounterVec
	invokeDuration       *prometheus.HistogramVec
	invokeErrorTotal     *prometheus.CounterVec
	eventDroppedTotal    *prometheus.CounterVec
	bulkheadQueued       *prometheus.GaugeVec
	bulkheadActive       *prometheus.GaugeVec
	bulkheadDroppedTotal *prometheus.CounterVec
	bulkheadWaitDuration *prometheus.HistogramVec
}

// NewCapabilityCollector creates a CapabilityCollector backed by stats.
// Returns a no-op collector when stats is nil or if registration fails.
func NewCapabilityCollector(st *stats.Stats) *CapabilityCollector {
	if st == nil {
		return &CapabilityCollector{}
	}
	var err error
	c := &CapabilityCollector{}
	c.invokeTotal, err = st.RegisterCounterVec("capability_invoke_total", "Invocations by capability, operation, and status", "capability", "operation", "status")
	if err != nil {
		log.Printf("[metrics] capability: failed to register counter vec: %v", err)
		return &CapabilityCollector{}
	}
	c.invokeDuration, err = st.RegisterHistogramVec("capability_invoke_duration_seconds", "Invocation duration distribution", "capability", "operation")
	if err != nil {
		log.Printf("[metrics] capability: failed to register histogram vec: %v", err)
		return &CapabilityCollector{}
	}
	c.invokeErrorTotal, err = st.RegisterCounterVec("capability_invoke_error_total", "Invocation errors by capability, operation, and error code", "capability", "operation", "error_code")
	if err != nil {
		log.Printf("[metrics] capability: failed to register counter vec: %v", err)
		return &CapabilityCollector{}
	}
	c.eventDroppedTotal, err = st.RegisterCounterVec("capability_event_dropped_total", "Events dropped due to pool overflow or shutdown", "capability", "operation", "reason")
	if err != nil {
		log.Printf("[metrics] capability: failed to register counter vec: %v", err)
		return &CapabilityCollector{}
	}
	c.bulkheadQueued, err = st.RegisterGaugeVec("capability_bulkhead_queued", "Invocations queued in bulkhead by capability", "capability")
	if err != nil {
		log.Printf("[metrics] capability: failed to register bulkhead_queued gauge: %v", err)
		return &CapabilityCollector{}
	}
	c.bulkheadActive, err = st.RegisterGaugeVec("capability_bulkhead_active", "Invocations active in bulkhead by capability", "capability")
	if err != nil {
		log.Printf("[metrics] capability: failed to register bulkhead_active gauge: %v", err)
		return &CapabilityCollector{}
	}
	c.bulkheadDroppedTotal, err = st.RegisterCounterVec("capability_bulkhead_dropped_total", "Invocations dropped by bulkhead by capability and reason", "capability", "reason")
	if err != nil {
		log.Printf("[metrics] capability: failed to register bulkhead_dropped counter: %v", err)
		return &CapabilityCollector{}
	}
	c.bulkheadWaitDuration, err = st.RegisterHistogramVec("capability_bulkhead_wait_seconds", "Bulkhead queue wait duration by capability", "capability")
	if err != nil {
		log.Printf("[metrics] capability: failed to register bulkhead_wait histogram: %v", err)
		return &CapabilityCollector{}
	}
	return c
}

// IncInvokeTotal increments the invoke counter for the given capability, operation, and status.
func (c *CapabilityCollector) IncInvokeTotal(capName, operation, status string) {
	if c.invokeTotal == nil {
		return
	}
	defer recoverLog("capability_invoke_total")
	c.invokeTotal.WithLabelValues(sanitizeLabel(capName), sanitizeLabel(operation), sanitizeLabel(status)).Inc()
}

// ObserveInvokeDuration records an invocation duration observation.
func (c *CapabilityCollector) ObserveInvokeDuration(capName, operation string, seconds float64) {
	if c.invokeDuration == nil {
		return
	}
	defer recoverLog("capability_invoke_duration_seconds")
	c.invokeDuration.WithLabelValues(sanitizeLabel(capName), sanitizeLabel(operation)).Observe(seconds)
}

// IncEventDropped increments the event dropped counter.
func (c *CapabilityCollector) IncEventDropped(capName, operation, reason string) {
	if c.eventDroppedTotal == nil {
		return
	}
	defer recoverLog("capability_event_dropped_total")
	c.eventDroppedTotal.WithLabelValues(sanitizeLabel(capName), sanitizeLabel(operation), sanitizeLabel(reason)).Inc()
}

// IncInvokeError increments the error counter for the given capability, operation, and error code.
func (c *CapabilityCollector) IncInvokeError(capName, operation, errorCode string) {
	if c.invokeErrorTotal == nil {
		return
	}
	defer recoverLog("capability_invoke_error_total")
	c.invokeErrorTotal.WithLabelValues(sanitizeLabel(capName), sanitizeLabel(operation), sanitizeLabel(errorCode)).Inc()
}

// IncBulkheadQueued increments the bulkhead queued gauge.
func (c *CapabilityCollector) IncBulkheadQueued(capName string) {
	if c.bulkheadQueued == nil {
		return
	}
	defer recoverLog("capability_bulkhead_queued")
	c.bulkheadQueued.WithLabelValues(sanitizeLabel(capName)).Inc()
}

// DecBulkheadQueued decrements the bulkhead queued gauge.
func (c *CapabilityCollector) DecBulkheadQueued(capName string) {
	if c.bulkheadQueued == nil {
		return
	}
	defer recoverLog("capability_bulkhead_queued")
	c.bulkheadQueued.WithLabelValues(sanitizeLabel(capName)).Dec()
}

// IncBulkheadActive increments the bulkhead active gauge.
func (c *CapabilityCollector) IncBulkheadActive(capName string) {
	if c.bulkheadActive == nil {
		return
	}
	defer recoverLog("capability_bulkhead_active")
	c.bulkheadActive.WithLabelValues(sanitizeLabel(capName)).Inc()
}

// DecBulkheadActive decrements the bulkhead active gauge.
func (c *CapabilityCollector) DecBulkheadActive(capName string) {
	if c.bulkheadActive == nil {
		return
	}
	defer recoverLog("capability_bulkhead_active")
	c.bulkheadActive.WithLabelValues(sanitizeLabel(capName)).Dec()
}

// IncBulkheadDropped increments the bulkhead dropped counter.
func (c *CapabilityCollector) IncBulkheadDropped(capName, reason string) {
	if c.bulkheadDroppedTotal == nil {
		return
	}
	defer recoverLog("capability_bulkhead_dropped_total")
	c.bulkheadDroppedTotal.WithLabelValues(sanitizeLabel(capName), sanitizeLabel(reason)).Inc()
}

// ObserveBulkheadWaitDuration records bulkhead queue wait duration.
func (c *CapabilityCollector) ObserveBulkheadWaitDuration(capName string, seconds float64) {
	if c.bulkheadWaitDuration == nil {
		return
	}
	defer recoverLog("capability_bulkhead_wait_seconds")
	c.bulkheadWaitDuration.WithLabelValues(sanitizeLabel(capName)).Observe(seconds)
}
