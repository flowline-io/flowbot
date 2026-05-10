package event

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEventConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "MessageSendEvent", value: types.MessageSendEvent},
		{name: "BotRunEvent", value: types.BotRunEvent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestActionFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   any
	}{
		{name: "SendMessage", fn: SendMessage},
		{name: "BotEventFire", fn: BotEventFire},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, tt.fn)
		})
	}
}
