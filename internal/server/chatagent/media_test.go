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

func TestResolveAttachmentsOwnership(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := chatagent.ResolveAttachments(ctx, "sess-a", "owner", []chatagent.AttachmentRef{
		{FileID: "missing"},
	})
	require.Error(t, err)
}
