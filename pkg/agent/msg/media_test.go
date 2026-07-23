package msg_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestKindFromMIME(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mime   string
		want   msg.MediaKind
		wantOK bool
	}{
		{name: "png image", mime: "image/png", want: msg.MediaKindImage, wantOK: true},
		{name: "jpeg with params", mime: "image/jpeg; charset=binary", want: msg.MediaKindImage, wantOK: true},
		{name: "wav audio", mime: "audio/wav", want: msg.MediaKindAudio, wantOK: true},
		{name: "mp4 video", mime: "video/mp4", want: msg.MediaKindVideo, wantOK: true},
		{name: "unknown", mime: "application/pdf", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := msg.KindFromMIME(tt.mime)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripMediaFromMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []msg.AgentMessage
		want int
	}{
		{
			name: "strips media keeps text",
			in: []msg.AgentMessage{msg.UserMessage{Parts: []msg.ContentPart{
				msg.TextPart{Text: "hi"},
				msg.MediaPart{Kind: msg.MediaKindImage, FileID: "f1"},
			}}},
			want: 1,
		},
		{
			name: "media only becomes stub text",
			in: []msg.AgentMessage{msg.UserMessage{Parts: []msg.ContentPart{
				msg.MediaPart{Kind: msg.MediaKindImage, FileID: "f1"},
			}}},
			want: 1,
		},
		{
			name: "assistant unchanged",
			in: []msg.AgentMessage{msg.AssistantMessage{Parts: []msg.ContentPart{
				msg.TextPart{Text: "ok"},
			}}},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := msg.StripMediaFromMessages(tt.in)
			assert.Len(t, out, len(tt.in))
			if user, ok := out[0].(msg.UserMessage); ok {
				assert.Len(t, user.Parts, tt.want)
				_, isMedia := user.Parts[0].(msg.MediaPart)
				assert.False(t, isMedia)
			}
		})
	}
}

func TestFilterMediaFromMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        []msg.AgentMessage
		keep      func(msg.MediaKind) bool
		wantKinds []msg.MediaKind
	}{
		{
			name: "keeps allowed kinds only",
			in: []msg.AgentMessage{msg.UserMessage{Parts: []msg.ContentPart{
				msg.TextPart{Text: "hi"},
				msg.MediaPart{Kind: msg.MediaKindImage, FileID: "i1"},
				msg.MediaPart{Kind: msg.MediaKindAudio, FileID: "a1"},
			}}},
			keep:      func(k msg.MediaKind) bool { return k == msg.MediaKindImage },
			wantKinds: []msg.MediaKind{msg.MediaKindImage},
		},
		{
			name: "keeps all when predicate always true",
			in: []msg.AgentMessage{msg.UserMessage{Parts: []msg.ContentPart{
				msg.MediaPart{Kind: msg.MediaKindVideo, FileID: "v1"},
			}}},
			keep:      func(msg.MediaKind) bool { return true },
			wantKinds: []msg.MediaKind{msg.MediaKindVideo},
		},
		{
			name: "drops all media to stub when none kept",
			in: []msg.AgentMessage{msg.UserMessage{Parts: []msg.ContentPart{
				msg.MediaPart{Kind: msg.MediaKindAudio, FileID: "a1"},
			}}},
			keep:      func(msg.MediaKind) bool { return false },
			wantKinds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := msg.FilterMediaFromMessages(tt.in, tt.keep)
			user, ok := out[0].(msg.UserMessage)
			assert.True(t, ok)
			var got []msg.MediaKind
			for _, part := range user.Parts {
				if mp, ok := part.(msg.MediaPart); ok {
					got = append(got, mp.Kind)
				}
			}
			assert.Equal(t, tt.wantKinds, got)
			if tt.wantKinds == nil {
				_, isText := user.Parts[0].(msg.TextPart)
				assert.True(t, isText)
			}
		})
	}
}
