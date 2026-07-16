package protocol_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestMessageSegmentBuilders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		segment  protocol.MessageSegment
		wantType string
		wantData map[string]any
	}{
		{
			name:     "text segment",
			segment:  protocol.Text("hello", " ", "world"),
			wantType: "text",
			wantData: map[string]any{"text": "hello world"},
		},
		{
			name:     "url segment",
			segment:  protocol.Url("https://example.com"),
			wantType: "url",
			wantData: map[string]any{"url": "https://example.com"},
		},
		{
			name:     "mention user",
			segment:  protocol.Mention("u-1"),
			wantType: "mention",
			wantData: map[string]any{"user_id": "u-1"},
		},
		{
			name:     "empty mention becomes mention_all",
			segment:  protocol.Mention(""),
			wantType: "mention_all",
			wantData: map[string]any{"user_id": "all"},
		},
		{
			name:     "mention all",
			segment:  protocol.MentionAll(),
			wantType: "mention_all",
			wantData: map[string]any{"user_id": "all"},
		},
		{
			name:     "image segment",
			segment:  protocol.Image("file-1"),
			wantType: "image",
			wantData: map[string]any{"file_id": "file-1"},
		},
		{
			name:     "voice segment",
			segment:  protocol.Voice("file-2"),
			wantType: "voice",
			wantData: map[string]any{"file_id": "file-2"},
		},
		{
			name:     "audio segment",
			segment:  protocol.Audio("file-3"),
			wantType: "audio",
			wantData: map[string]any{"file_id": "file-3"},
		},
		{
			name:     "video segment",
			segment:  protocol.Video("file-4"),
			wantType: "video",
			wantData: map[string]any{"file_id": "file-4"},
		},
		{
			name:     "file segment",
			segment:  protocol.File("file-5"),
			wantType: "file",
			wantData: map[string]any{"file_id": "file-5"},
		},
		{
			name:     "location segment",
			segment:  protocol.Location(1.5, 2.5, "HQ", "office"),
			wantType: "location",
			wantData: map[string]any{
				"latitude":  1.5,
				"longitude": 2.5,
				"title":     "HQ",
				"content":   "office",
			},
		},
		{
			name:     "reply segment",
			segment:  protocol.Reply("u-2", "m-9"),
			wantType: "reply",
			wantData: map[string]any{"user_id": "u-2", "message_id": "m-9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantType, tt.segment.Type)
			assert.Equal(t, tt.wantData, tt.segment.Data)
			assert.NotEmpty(t, tt.segment.String())
		})
	}
}

func TestMessageString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message protocol.Message
		wantSub string
	}{
		{
			name:    "single text message",
			message: protocol.Message{protocol.Text("hi")},
			wantSub: "text",
		},
		{
			name:    "multi segment message",
			message: protocol.Message{protocol.Text("a"), protocol.Url("https://x.test")},
			wantSub: "url",
		},
		{
			name:    "empty message",
			message: protocol.Message{},
			wantSub: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.message.String()
			if tt.wantSub == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tt.wantSub)
		})
	}
}
