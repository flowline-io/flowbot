package chatagent

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestTokenUsageSourceFromRunKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind RunKind
		want string
	}{
		{name: "interactive agent", kind: RunKindInteractive, want: types.TokenUsageSourceAgent},
		{name: "pipeline", kind: RunKindPipeline, want: types.TokenUsageSourcePipeline},
		{name: "scheduled task", kind: RunKindScheduled, want: types.TokenUsageSourceScheduledTask},
		{name: "empty kind defaults agent", kind: "", want: types.TokenUsageSourceAgent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TokenUsageSourceFromRunKind(tt.kind))
		})
	}
}
