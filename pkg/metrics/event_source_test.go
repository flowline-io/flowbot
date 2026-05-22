package metrics

import (
	"testing"

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
