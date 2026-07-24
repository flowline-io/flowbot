package msg_test

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestEnsureToolCallID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		keep  bool
	}{
		{name: "preserves nonempty", input: "call_abc", keep: true},
		{name: "synthesizes empty", input: "", keep: false},
		{name: "synthesizes whitespace", input: "   ", keep: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := msg.EnsureToolCallID(tt.input)
			if tt.keep {
				assert.Equal(t, tt.input, got)
				return
			}
			assert.True(t, strings.HasPrefix(got, "call_"))
			assert.Greater(t, len(got), len("call_"))
		})
	}
}
