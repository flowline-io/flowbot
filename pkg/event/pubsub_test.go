package event

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestPublishMessage(t *testing.T) {
	// This test verifies that PublishMessage handles nil publisher gracefully
	// In production, Publisher should be initialized via NewPublisher

	// Skip this test as it requires Redis connection
	t.Skip("requires Redis connection and initialized Publisher")
}

func TestMessageTypes(t *testing.T) {
	// Test that message types are properly structured
	msg := types.Message{
		Platform: "slack",
		Topic:    "general",
		Payload: types.EventPayload{
			Typ: "text",
			Src: []byte("test message"),
		},
	}

	assert.Equal(t, "slack", msg.Platform)
	assert.Equal(t, "general", msg.Topic)
	assert.Equal(t, "text", msg.Payload.Typ)
}

func TestBotEventType(t *testing.T) {
	// Test that bot event type is properly structured
	event := types.BotEvent{
		EventName: "reminder",
		Uid:       "user123",
		Topic:     "personal",
		Param: types.KV{
			"message": "Don't forget the meeting",
		},
	}

	assert.Equal(t, "reminder", event.EventName)
	assert.Equal(t, "user123", event.Uid)
	assert.Equal(t, "personal", event.Topic)
	assert.Equal(t, "Don't forget the meeting", event.Param["message"])
}
