package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamProgressTrackerRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		recordReasoning []string
		recordText      []string
		wantReasoning   int
		wantText        int
	}{
		{
			name:            "empty deltas ignored",
			recordReasoning: []string{"", ""},
			wantReasoning:   0,
		},
		{
			name:            "accumulates reasoning chars",
			recordReasoning: []string{"think", "ing"},
			wantReasoning:   8,
		},
		{
			name:       "accumulates text chars",
			recordText: []string{"hi", " there"},
			wantText:   8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracker := newStreamProgressTracker("test-model", time.Minute, nil)
			for _, delta := range tt.recordReasoning {
				tracker.recordReasoning(delta)
			}
			for _, delta := range tt.recordText {
				tracker.recordText(delta)
			}
			assert.Equal(t, tt.wantReasoning, tracker.reasoningCharsForTest())
			assert.Equal(t, tt.wantText, tracker.textCharsForTest())
		})
	}
}

func TestStreamProgressTrackerIdleTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		idle      time.Duration
		wait      time.Duration
		wantIdle  bool
		record    string
		tickFinal bool
	}{
		{
			name:     "cancels stalled stream",
			idle:     20 * time.Millisecond,
			wait:     25 * time.Millisecond,
			wantIdle: true,
			record:   "stall",
		},
		{
			name:     "no cancel before idle limit",
			idle:     200 * time.Millisecond,
			wait:     10 * time.Millisecond,
			wantIdle: false,
			record:   "fresh",
		},
		{
			name:      "final tick does not cancel",
			idle:      20 * time.Millisecond,
			wait:      25 * time.Millisecond,
			wantIdle:  false,
			record:    "done",
			tickFinal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancelCause(context.Background())
			t.Cleanup(func() { cancel(nil) })

			tracker := newStreamProgressTracker("test-model", tt.idle, cancel)
			tracker.recordReasoning(tt.record)
			tracker.begin(ctx)
			time.Sleep(tt.wait)
			tracker.tick(tt.tickFinal)

			if tt.wantIdle {
				require.ErrorIs(t, context.Cause(ctx), ErrStreamIdle)
				return
			}
			assert.NotErrorIs(t, context.Cause(ctx), ErrStreamIdle)
		})
	}
}

func TestStreamProgressTrackerEndWithoutDeltas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action func(*streamProgressTracker)
	}{
		{
			name:   "end without deltas is safe",
			action: func(tr *streamProgressTracker) { tr.end() },
		},
		{
			name: "end after record without begin",
			action: func(tr *streamProgressTracker) {
				tr.recordText("x")
				tr.end()
			},
		},
		{
			name: "double end is safe",
			action: func(tr *streamProgressTracker) {
				tr.end()
				tr.end()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracker := newStreamProgressTracker("test-model", time.Minute, nil)
			tt.action(tracker)
		})
	}
}
