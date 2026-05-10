package event

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestPublishMessage(t *testing.T) {
	t.Parallel()
	t.Skip("requires Redis connection and initialized Publisher")
}

func TestMessageStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     types.Message
		wantPl  string
		wantTpc string
		wantTyp string
	}{
		{
			name: "basic message",
			msg: types.Message{
				Platform: "slack",
				Topic:    "general",
				Payload: types.EventPayload{
					Typ: "text",
					Src: []byte("test message"),
				},
			},
			wantPl:  "slack",
			wantTpc: "general",
			wantTyp: "text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantPl, tt.msg.Platform)
			assert.Equal(t, tt.wantTpc, tt.msg.Topic)
			assert.Equal(t, tt.wantTyp, tt.msg.Payload.Typ)
		})
	}
}

func TestBotEventStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		event     types.BotEvent
		wantName  string
		wantUID   string
		wantTopic string
		wantMsg   string
	}{
		{
			name: "bot event with params",
			event: types.BotEvent{
				EventName: "reminder",
				Uid:       "user123",
				Topic:     "personal",
				Param: types.KV{
					"message": "Don't forget the meeting",
				},
			},
			wantName:  "reminder",
			wantUID:   "user123",
			wantTopic: "personal",
			wantMsg:   "Don't forget the meeting",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantName, tt.event.EventName)
			assert.Equal(t, tt.wantUID, tt.event.Uid)
			assert.Equal(t, tt.wantTopic, tt.event.Topic)
			assert.Equal(t, tt.wantMsg, tt.event.Param["message"])
		})
	}
}
