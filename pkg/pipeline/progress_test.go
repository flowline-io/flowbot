package pipeline

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTerminalProgressJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		evt  StepProgressEvent
		want bool
	}{
		{
			name: "complete is terminal",
			evt:  StepProgressEvent{StepIndex: -1, Status: "complete"},
			want: true,
		},
		{
			name: "failed is terminal",
			evt:  StepProgressEvent{StepIndex: -1, Status: "failed"},
			want: true,
		},
		{
			name: "start is not terminal",
			evt:  StepProgressEvent{StepIndex: -1, Status: "start"},
			want: false,
		},
		{
			name: "step done is not terminal",
			evt: StepProgressEvent{
				StepIndex: 0, Status: "done",
				Output: map[string]any{"blob": string(make([]byte, 1024))},
			},
			want: false,
		},
		{
			name: "running is not terminal",
			evt:  StepProgressEvent{StepIndex: 2, Status: "running"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw, err := sonic.MarshalString(tt.evt)
			require.NoError(t, err)
			assert.Equal(t, tt.want, IsTerminalProgressJSON(raw))
		})
	}
	assert.False(t, IsTerminalProgressJSON("{"))
	assert.False(t, IsTerminalProgressJSON(`{"status":"complete"}`))
}
