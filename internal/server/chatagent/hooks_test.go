package chatagent

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/stretchr/testify/assert"
)

func TestRegisterHooksObservesEvents(t *testing.T) {
	tests := []struct {
		name      string
		event     hooks.ObservationEvent
		wantCalls int32
	}{
		{
			name:      "save point observed",
			event:     hooks.ObservationEvent{Type: hooks.EventSavePoint},
			wantCalls: 1,
		},
		{
			name: "context usage observed",
			event: hooks.ObservationEvent{
				Type: hooks.EventContextUsage,
				ContextUsage: &hooks.ContextUsageInfo{
					Tokens:        100,
					ContextWindow: 1000,
					Percent:       10,
				},
			},
			wantCalls: 1,
		},
		{
			name:      "model update observed",
			event:     hooks.ObservationEvent{Type: hooks.EventModelUpdate, ModelName: "gpt"},
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			var calls atomic.Int32
			hooks.Observe(reg, func(context.Context, hooks.ObservationEvent) error {
				calls.Add(1)
				return nil
			})
			RegisterHooks(reg, ChatHookDeps{SessionID: "sess-1"})
			reg.EmitObservation(context.Background(), tt.event, nil)
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}
