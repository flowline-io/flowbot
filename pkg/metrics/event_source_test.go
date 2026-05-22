package metrics

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewEventSourceCollector(t *testing.T) {
	tests := []struct {
		name  string
		st    *stats.Stats
		isNil bool
	}{
		{
			name:  "nil stats returns no-op collector",
			st:    nil,
			isNil: false,
		},
		{
			name:  "valid stats returns functional collector",
			st:    stats.NewStats(),
			isNil: false,
		},
		{
			name:  "reuse stats instance",
			st:    stats.NewStats(),
			isNil: false,
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
}
