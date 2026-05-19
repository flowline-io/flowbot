package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewEventCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewEventCollector(nil)
		assert.NotNil(t, c)
		c.IncReceived("bookmark.created", "ability")
		c.IncMatched("bookmark.created", "archive-items")
		c.IncDedup("bookmark.created", "archive-items")
		c.ObserveLag("bookmark.created", 0.5)
	})
}

func TestEventCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewEventCollector(s)

	c.IncReceived("bookmark.created", "ability")
	c.IncReceived("bookmark.created", "ability")
	c.IncReceived("kanban.task.created", "ability")

	c.IncMatched("bookmark.created", "archive-items")
	c.IncMatched("bookmark.created", "sync-bookmarks")

	c.IncDedup("bookmark.created", "archive-items")

	c.ObserveLag("bookmark.created", 1.2)

	expected := `
# HELP event_received_total Events received by event type and source
# TYPE event_received_total counter
event_received_total{event_type="bookmark.created",source="ability"} 2
event_received_total{event_type="kanban.task.created",source="ability"} 1
`
	err := testutil.CollectAndCompare(c.receivedTotal, strings.NewReader(expected), "event_received_total")
	assert.NoError(t, err)
}

func TestEventCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewEventCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncReceived", fn: func() { c.IncReceived("e", "s") }},
		{name: "IncMatched", fn: func() { c.IncMatched("e", "p") }},
		{name: "IncDedup", fn: func() { c.IncDedup("e", "p") }},
		{name: "ObserveLag", fn: func() { c.ObserveLag("e", 1.0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
