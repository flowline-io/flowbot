package model_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestSupportsModality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		kind      msg.MediaKind
		want      bool
	}{
		{name: "known vision model image", modelName: "gpt-5.3-codex", kind: msg.MediaKindImage, want: true},
		{name: "known text-only model image", modelName: "deepseek-v4-pro", kind: msg.MediaKindImage, want: false},
		{name: "unknown model allows image", modelName: "custom-vision", kind: msg.MediaKindImage, want: true},
		{name: "unknown model rejects audio", modelName: "custom-vision", kind: msg.MediaKindAudio, want: false},
		{name: "known model rejects video until catalog", modelName: "gpt-5.5-pro", kind: msg.MediaKindVideo, want: false},
		{name: "mimo v2.5 accepts video", modelName: "mimo-v2.5", kind: msg.MediaKindVideo, want: true},
		{name: "mimo v2.5 pro accepts audio", modelName: "mimo-v2.5-pro", kind: msg.MediaKindAudio, want: true},
		{name: "grok 4.5 rejects image", modelName: "grok-4.5", kind: msg.MediaKindImage, want: false},
		{name: "unknown media kind rejected", modelName: "mimo-v2.5", kind: msg.MediaKind("file"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, model.SupportsModality(tt.modelName, tt.kind))
		})
	}
}
