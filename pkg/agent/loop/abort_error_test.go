package loop

import (
	"context"
	"fmt"
	"testing"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbortLoopError(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want []error
	}{
		{
			name: "llm aborted becomes loop aborted",
			in:   agentllm.ErrAborted,
			want: []error{ErrAborted, agentllm.ErrAborted},
		},
		{
			name: "preserves deadline cause",
			in:   fmt.Errorf("%w: %w", agentllm.ErrAborted, context.DeadlineExceeded),
			want: []error{ErrAborted, agentllm.ErrAborted, context.DeadlineExceeded},
		},
		{
			name: "preserves cancel cause",
			in:   fmt.Errorf("%w: %w", agentllm.ErrAborted, context.Canceled),
			want: []error{ErrAborted, context.Canceled},
		},
		{
			name: "already loop aborted",
			in:   fmt.Errorf("%w: %w", ErrAborted, context.Canceled),
			want: []error{ErrAborted, context.Canceled},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := abortLoopError(tt.in)
			for _, want := range tt.want {
				require.ErrorIs(t, got, want)
			}
		})
	}

	t.Run("non abort unchanged", func(t *testing.T) {
		t.Parallel()
		in := assert.AnError
		assert.Equal(t, in, abortLoopError(in))
	})
}
