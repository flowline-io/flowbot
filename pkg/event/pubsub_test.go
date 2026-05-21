package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		{
			name:    "empty message",
			msg:     types.Message{},
			wantPl:  "",
			wantTpc: "",
			wantTyp: "",
		},
		{
			name: "message with binary payload",
			msg: types.Message{
				Platform: "discord",
				Topic:    "bot-log",
				Payload: types.EventPayload{
					Typ: "binary",
					Src: []byte{0x01, 0x02, 0x03},
				},
			},
			wantPl:  "discord",
			wantTpc: "bot-log",
			wantTyp: "binary",
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

func TestNewRouterSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "NewRouter creates router with nil TracerProvider"},
		{name: "NewRouter returns valid router"},
		{name: "NewRouter router is closable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router, err := NewRouter(nil)
			require.NoError(t, err)
			require.NotNil(t, router)
			_ = router.Close()
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
		wantMsg   any
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
		{
			name: "empty bot event",
			event: types.BotEvent{
				Param: types.KV{},
			},
			wantName:  "",
			wantUID:   "",
			wantTopic: "",
			wantMsg:   nil,
		},
		{
			name: "bot event with multiple params",
			event: types.BotEvent{
				EventName: "notify",
				Uid:       "user456",
				Topic:     "alerts",
				Param: types.KV{
					"message":  "Server CPU > 90%",
					"severity": "critical",
					"action":   "scale_up",
				},
			},
			wantName:  "notify",
			wantUID:   "user456",
			wantTopic: "alerts",
			wantMsg:   "Server CPU > 90%",
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
