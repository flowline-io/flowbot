package event

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestMessageTypeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "MessageSendEvent", got: types.MessageSendEvent, want: "message:send"},
		{name: "BotRunEvent", got: types.BotRunEvent, want: "bot:event"},
		{name: "InstructPushEvent", got: types.InstructPushEvent, want: "instruct:push"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestEventPayloadStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload types.EventPayload
		wantTyp string
		wantSrc []byte
	}{
		{
			name: "text payload",
			payload: types.EventPayload{
				Typ: "text",
				Src: []byte("test data"),
			},
			wantTyp: "text",
			wantSrc: []byte("test data"),
		},
		{
			name:    "empty payload",
			payload: types.EventPayload{},
			wantTyp: "",
			wantSrc: nil,
		},
		{
			name: "binary payload",
			payload: types.EventPayload{
				Typ: "binary",
				Src: []byte{0x00, 0xFF, 0xAB},
			},
			wantTyp: "binary",
			wantSrc: []byte{0x00, 0xFF, 0xAB},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantTyp, tt.payload.Typ)
			assert.Equal(t, tt.wantSrc, tt.payload.Src)
		})
	}
}
