package chatagent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMIMEAllowlist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mime    string
		want    msg.MediaKind
		wantErr bool
	}{
		{name: "png", mime: "image/png", want: msg.MediaKindImage},
		{name: "mp4 video", mime: "video/mp4", want: msg.MediaKindVideo},
		{name: "pdf rejected", mime: "application/pdf", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := chatagent.ValidateMIMEAllowlist(tt.mime)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRejectUnsupportedModalities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		parts     []msg.ContentPart
		wantErr   bool
	}{
		{
			name:      "vision model accepts image",
			modelName: "gpt-5.3-codex",
			parts:     []msg.ContentPart{msg.MediaPart{Kind: msg.MediaKindImage}},
		},
		{
			name:      "text model rejects image",
			modelName: "deepseek-v4-pro",
			parts:     []msg.ContentPart{msg.MediaPart{Kind: msg.MediaKindImage}},
			wantErr:   true,
		},
		{
			name:      "audio rejected until catalog",
			modelName: "gpt-5.5-pro",
			parts:     []msg.ContentPart{msg.MediaPart{Kind: msg.MediaKindAudio}},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := chatagent.RejectUnsupportedModalities(tt.modelName, tt.parts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMediaDataURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mime string
		data []byte
		want string
	}{
		{
			name: "png bytes",
			mime: "image/png",
			data: []byte{0x89, 0x50, 0x4e, 0x47},
			want: "data:image/png;base64,iVBORw==",
		},
		{
			name: "empty mime falls back",
			mime: "",
			data: []byte{1, 2, 3},
			want: "data:application/octet-stream;base64,AQID",
		},
		{
			name: "jpeg mime preserved",
			mime: "image/jpeg",
			data: []byte{0xff, 0xd8},
			want: "data:image/jpeg;base64,/9g=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, chatagent.MediaDataURI(tt.mime, tt.data))
		})
	}
}

func TestFillMediaPartForProvider(t *testing.T) {
	t.Parallel()

	png := []byte{0x89, 0x50, 0x4e, 0x47}
	tests := []struct {
		name     string
		provider string
		kind     msg.MediaKind
		mime     string
		data     []byte
		wantURL  string
		wantData bool
	}{
		{
			name:     "openai compatible image uses data uri",
			provider: "openai_compatible",
			kind:     msg.MediaKindImage,
			mime:     "image/png",
			data:     png,
			wantURL:  "data:image/png;base64,iVBORw==",
		},
		{
			name:     "openai image uses data uri",
			provider: "openai",
			kind:     msg.MediaKindImage,
			mime:     "image/png",
			data:     png,
			wantURL:  "data:image/png;base64,iVBORw==",
		},
		{
			name:     "anthropic image keeps binary",
			provider: "anthropic",
			kind:     msg.MediaKindImage,
			mime:     "image/png",
			data:     png,
			wantData: true,
		},
		{
			name:     "default image prefers data uri over private signed url",
			provider: "unknown",
			kind:     msg.MediaKindImage,
			mime:     "image/png",
			data:     png,
			wantURL:  "data:image/png;base64,iVBORw==",
		},
		{
			name:     "audio always binary",
			provider: "openai_compatible",
			kind:     msg.MediaKindAudio,
			mime:     "audio/wav",
			data:     []byte{1, 2},
			wantData: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := chatagent.FillMediaPartForProvider(tt.provider, msg.MediaPart{
				Kind:     tt.kind,
				MIMEType: tt.mime,
				FileID:   "f1",
			}, tt.data)
			if tt.wantData {
				assert.Equal(t, tt.data, got.Data)
				assert.Empty(t, got.URL)
				return
			}
			assert.Equal(t, tt.wantURL, got.URL)
			assert.Nil(t, got.Data)
		})
	}
}

func TestResolveAttachmentsOwnership(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := chatagent.ResolveAttachments(ctx, "sess-a", "owner", []chatagent.AttachmentRef{
		{FileID: "missing"},
	})
	require.Error(t, err)
}
