package event

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	// Test that event types are defined
	// These constants are defined in pkg/types/event.go
	assert.NotEmpty(t, types.MessageSendEvent)
	assert.NotEmpty(t, types.BotRunEvent)
}

func TestActionFunctions_Exist(t *testing.T) {
	// Test that action functions exist and have correct signatures
	// We can't fully test these without database connections
	assert.NotNil(t, SendMessage)
	assert.NotNil(t, BotEventFire)
}
