package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewEventSourceCollector(t *testing.T) {
	tests := []struct {
		name string
		st   *stats.Stats
	}{
		{
			name: "nil stats returns no-op collector",
			st:   nil,
		},
		{
			name: "valid stats returns functional collector",
			st:   stats.NewStats(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewEventSourceCollector(tt.st)
			if c == nil {
				t.Fatal("NewEventSourceCollector returned nil")
			}
			c.IncPollTotal("test/rsrc", "success")
			c.IncPollEvents("test/rsrc", "created")
			c.ObservePollDuration("test/rsrc", 0.1)
			c.IncPollError("test/rsrc")
			c.IncWebhookTotal("github/events", "202")
			c.IncWebhookEvents("github/events")
			c.ObserveStateFlushDuration(0.05)
		})
	}

	t.Run("two collectors from same stats instance do not panic", func(t *testing.T) {
		st := stats.NewStats()
		c1 := NewEventSourceCollector(st)
		if c1 == nil {
			t.Fatal("NewEventSourceCollector returned nil for c1")
		}
		c2 := NewEventSourceCollector(st)
		if c2 == nil {
			t.Fatal("NewEventSourceCollector returned nil for c2")
		}
		c1.IncPollTotal("rsrc_a", "success")
		c2.IncPollTotal("rsrc_b", "error")
		c1.IncPollEvents("rsrc_a", "created")
		c2.ObservePollDuration("rsrc_b", 0.5)
		c1.ObserveStateFlushDuration(0.1)
		c2.IncWebhookTotal("gitlab/events", "200")
		c1.IncWebhookEvents("github/events")
	})
}

func TestEventSourceCollector_CounterMetrics(t *testing.T) {
	pollTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "event_source_poll_total_test"},
		[]string{"resource", "status"},
	)
	webhookTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "event_source_webhook_total_test"},
		[]string{"resource", "status"},
	)
	pollEvents := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "event_source_poll_events_total_test"},
		[]string{"resource", "event_type"},
	)
	pollError := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "event_source_poll_error_total_test"},
		[]string{"resource"},
	)
	stateFlush := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "event_source_state_flush_duration_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"operation"},
	)
	c := &EventSourceCollector{
		pollTotal:      pollTotal,
		webhookTotal:   webhookTotal,
		pollEvents:     pollEvents,
		pollErrorTotal: pollError,
		stateFlushDur:  stateFlush,
	}

	c.IncPollTotal("rss/feed", "success")
	c.IncPollTotal("rss/feed", "error")
	c.IncPollEvents("rss/feed", "item.created")
	c.IncWebhookTotal("github/hooks", "202")
	c.IncWebhookEvents("github/hooks")
	c.IncPollError("rss/feed")
	c.ObservePollDuration("rss/feed", 0.3)
	c.ObserveStateFlushDuration(0.1)

	expected := `
# HELP event_source_poll_total_test
# TYPE event_source_poll_total_test counter
event_source_poll_total_test{resource="rss_feed",status="error"} 1
event_source_poll_total_test{resource="rss_feed",status="success"} 1
`
	err := testutil.CollectAndCompare(pollTotal, strings.NewReader(expected), "event_source_poll_total_test")
	assert.NoError(t, err)
}
