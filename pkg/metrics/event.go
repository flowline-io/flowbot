package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// EventCollector holds typed metrics for event processing.
// When initialized with a nil stats, all methods are no-op.
type EventCollector struct {
	receivedTotal *prometheus.CounterVec
	matchedTotal  *prometheus.CounterVec
	dedupTotal    *prometheus.CounterVec
	lagSeconds    *prometheus.HistogramVec
}

// NewEventCollector creates an EventCollector backed by stats.
// Returns a no-op collector when stats is nil or if registration fails.
func NewEventCollector(st *stats.Stats) *EventCollector {
	if st == nil {
		return &EventCollector{}
	}
	var err error
	c := &EventCollector{}
	c.receivedTotal, err = st.RegisterCounterVec("event_received_total", "Events received by event type and source", "event_type", "source")
	if err != nil {
		log.Printf("[metrics] event: failed to register counter vec: %v", err)
		return &EventCollector{}
	}
	c.matchedTotal, err = st.RegisterCounterVec("event_matched_total", "Events matched to a pipeline", "event_type", "pipeline")
	if err != nil {
		log.Printf("[metrics] event: failed to register counter vec: %v", err)
		return &EventCollector{}
	}
	c.dedupTotal, err = st.RegisterCounterVec("event_dedup_total", "Idempotent consumption filter hits", "event_type", "pipeline")
	if err != nil {
		log.Printf("[metrics] event: failed to register counter vec: %v", err)
		return &EventCollector{}
	}
	c.lagSeconds, err = st.RegisterHistogramVec("event_lag_seconds", "Delay from event creation to consumption", "event_type")
	if err != nil {
		log.Printf("[metrics] event: failed to register histogram vec: %v", err)
		return &EventCollector{}
	}
	return c
}

// IncReceived increments the received counter for the given event type and source.
func (c *EventCollector) IncReceived(eventType, source string) {
	if c.receivedTotal == nil {
		return
	}
	defer recoverLog("event_received_total")
	c.receivedTotal.WithLabelValues(sanitizeLabel(eventType), sanitizeLabel(source)).Inc()
}

// IncMatched increments the matched counter for the given event type and pipeline.
func (c *EventCollector) IncMatched(eventType, pipeline string) {
	if c.matchedTotal == nil {
		return
	}
	defer recoverLog("event_matched_total")
	c.matchedTotal.WithLabelValues(sanitizeLabel(eventType), sanitizeLabel(pipeline)).Inc()
}

// IncDedup increments the dedup counter for the given event type and pipeline.
func (c *EventCollector) IncDedup(eventType, pipeline string) {
	if c.dedupTotal == nil {
		return
	}
	defer recoverLog("event_dedup_total")
	c.dedupTotal.WithLabelValues(sanitizeLabel(eventType), sanitizeLabel(pipeline)).Inc()
}

// ObserveLag records a lag observation in seconds for the given event type.
func (c *EventCollector) ObserveLag(eventType string, seconds float64) {
	if c.lagSeconds == nil {
		return
	}
	defer recoverLog("event_lag_seconds")
	c.lagSeconds.WithLabelValues(sanitizeLabel(eventType)).Observe(seconds)
}
