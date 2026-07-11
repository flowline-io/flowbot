package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliverRunResultDoneDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   int64
	}{
		{name: "short run", ms: 250},
		{name: "multi second run", ms: 3200},
		{name: "sub second run", ms: 800},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := &apiEventRecorder{}
			deliverRunResult(context.Background(), nil, RunRequest{
				API: &APIRunOptions{Publisher: pub},
			}, "ok", nil, nil, nil, time.Duration(tt.ms)*time.Millisecond)

			require.Len(t, pub.events, 1)
			assert.Equal(t, EventTypeDone, pub.events[0].Type)
			assert.Equal(t, int64(tt.ms), pub.events[0].DurationMs)
			assert.Equal(t, "ok", pub.events[0].Text)
		})
	}
}
