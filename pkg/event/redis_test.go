package event

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestMessageTypes_Constants(t *testing.T) {
	// Test event type constants
	assert.Equal(t, "message:send", types.MessageSendEvent)
	assert.Equal(t, "bot:event", types.BotRunEvent)
}

func TestEventPayload(t *testing.T) {
	payload := types.EventPayload{
		Typ: "text",
		Src: []byte("test data"),
	}

	assert.Equal(t, "text", payload.Typ)
	assert.Equal(t, []byte("test data"), payload.Src)
}
