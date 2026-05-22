package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// EventSourceCollector holds typed metrics for the provider event source system.
// When initialized with a nil stats, all methods are no-op.
type EventSourceCollector struct {
	pollTotal      *prometheus.CounterVec
	pollEvents     *prometheus.CounterVec
	pollDuration   *prometheus.HistogramVec
	pollErrorTotal *prometheus.CounterVec
	webhookTotal   *prometheus.CounterVec
	webhookEvents  *prometheus.CounterVec
	stateFlushDur  *prometheus.HistogramVec
}

// NewEventSourceCollector creates an EventSourceCollector backed by stats.
// Returns a no-op collector when stats is nil or if registration fails.
func NewEventSourceCollector(st *stats.Stats) *EventSourceCollector {
	if st == nil {
		return &EventSourceCollector{}
	}
	var err error
	c := &EventSourceCollector{}
	c.pollTotal, err = st.RegisterCounterVec("event_source_poll_total", "Poll completions by resource and status", "resource", "status")
	if err != nil {
		log.Printf("[metrics] event_source: poll_total: %v", err)
		return &EventSourceCollector{}
	}
	c.pollEvents, err = st.RegisterCounterVec("event_source_poll_events_total", "Events emitted per poll by resource and event type", "resource", "event_type")
	if err != nil {
		log.Printf("[metrics] event_source: poll_events: %v", err)
		return &EventSourceCollector{}
	}
	c.pollDuration, err = st.RegisterHistogramVec("event_source_poll_duration_seconds", "Poll execution time by resource", "resource")
	if err != nil {
		log.Printf("[metrics] event_source: poll_duration: %v", err)
		return &EventSourceCollector{}
	}
	c.pollErrorTotal, err = st.RegisterCounterVec("event_source_poll_error_total", "Failed polls by resource", "resource")
	if err != nil {
		log.Printf("[metrics] event_source: poll_error: %v", err)
		return &EventSourceCollector{}
	}
	c.webhookTotal, err = st.RegisterCounterVec("event_source_webhook_total", "Webhook requests by path and status", "path", "status")
	if err != nil {
		log.Printf("[metrics] event_source: webhook_total: %v", err)
		return &EventSourceCollector{}
	}
	c.webhookEvents, err = st.RegisterCounterVec("event_source_webhook_events_total", "Events emitted per webhook by path", "path")
	if err != nil {
		log.Printf("[metrics] event_source: webhook_events: %v", err)
		return &EventSourceCollector{}
	}
	c.stateFlushDur, err = st.RegisterHistogramVec("event_source_state_flush_duration_seconds", "PG state flush duration", "operation")
	if err != nil {
		log.Printf("[metrics] event_source: state_flush_duration: %v", err)
		return &EventSourceCollector{}
	}
	return c
}

// IncPollTotal increments the poll completion counter.
func (c *EventSourceCollector) IncPollTotal(resource, status string) {
	if c.pollTotal == nil {
		return
	}
	defer recoverLog("event_source_poll_total")
	c.pollTotal.WithLabelValues(sanitizeLabel(resource), sanitizeLabel(status)).Inc()
}

// IncPollEvents increments the poll events counter.
func (c *EventSourceCollector) IncPollEvents(resource, eventType string) {
	if c.pollEvents == nil {
		return
	}
	defer recoverLog("event_source_poll_events_total")
	c.pollEvents.WithLabelValues(sanitizeLabel(resource), sanitizeLabel(eventType)).Inc()
}

// ObservePollDuration records the poll execution time in seconds.
func (c *EventSourceCollector) ObservePollDuration(resource string, seconds float64) {
	if c.pollDuration == nil {
		return
	}
	defer recoverLog("event_source_poll_duration_seconds")
	c.pollDuration.WithLabelValues(sanitizeLabel(resource)).Observe(seconds)
}

// IncPollError increments the poll error counter.
func (c *EventSourceCollector) IncPollError(resource string) {
	if c.pollErrorTotal == nil {
		return
	}
	defer recoverLog("event_source_poll_error_total")
	c.pollErrorTotal.WithLabelValues(sanitizeLabel(resource)).Inc()
}

// IncWebhookTotal increments the webhook request counter.
func (c *EventSourceCollector) IncWebhookTotal(path, status string) {
	if c.webhookTotal == nil {
		return
	}
	defer recoverLog("event_source_webhook_total")
	c.webhookTotal.WithLabelValues(sanitizeLabel(path), sanitizeLabel(status)).Inc()
}

// IncWebhookEvents increments the webhook events counter.
func (c *EventSourceCollector) IncWebhookEvents(path string) {
	if c.webhookEvents == nil {
		return
	}
	defer recoverLog("event_source_webhook_events_total")
	c.webhookEvents.WithLabelValues(sanitizeLabel(path)).Inc()
}

// ObserveStateFlushDuration records the PG state flush duration in seconds.
func (c *EventSourceCollector) ObserveStateFlushDuration(seconds float64) {
	if c.stateFlushDur == nil {
		return
	}
	defer recoverLog("event_source_state_flush_duration_seconds")
	c.stateFlushDur.WithLabelValues("flush").Observe(seconds)
}
