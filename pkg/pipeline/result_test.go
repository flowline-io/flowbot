package pipeline

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
)

func TestStepResultFromInvoke(t *testing.T) {
	tests := []struct {
		name string
		res  *ability.InvokeResult
		want map[string]any
	}{
		{
			name: "nil result",
			res:  nil,
			want: map[string]any{},
		},
		{
			name: "map data passthrough",
			res: &ability.InvokeResult{
				Capability: hub.CapAgent,
				Operation:  "run",
				Data:       map[string]any{"reply": "hello", "session_id": "s1"},
			},
			want: map[string]any{"reply": "hello", "session_id": "s1"},
		},
		{
			name: "scalar data wrapped",
			res: &ability.InvokeResult{
				Data: "plain",
			},
			want: map[string]any{"items": "plain"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepResultFromInvoke(tt.res)
			assert.Equal(t, tt.want, got)
		})
	}
}
