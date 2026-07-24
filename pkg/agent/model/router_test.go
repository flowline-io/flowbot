package model_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_Select(t *testing.T) {
	tests := []struct {
		name               string
		chat               string
		tool               string
		afterToolExecution bool
		want               string
	}{
		{name: "chat by default", chat: "chat-model", tool: "tool-model", afterToolExecution: false, want: "chat-model"},
		{name: "tool after execution", chat: "chat-model", tool: "tool-model", afterToolExecution: true, want: "tool-model"},
		{name: "fallback to tool", chat: "", tool: "tool-model", afterToolExecution: false, want: "tool-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := model.NewRouter(tt.chat, tt.tool)
			assert.Equal(t, tt.want, router.Select(tt.afterToolExecution))
		})
	}
}

func TestRouter_ApplyToContext(t *testing.T) {
	tests := []struct {
		name               string
		afterToolExecution bool
		wantModel          string
	}{
		{name: "updates model on context", afterToolExecution: true, wantModel: "tool"},
		{name: "uses chat model by default", afterToolExecution: false, wantModel: "chat"},
		{name: "handles nil context safely", afterToolExecution: false, wantModel: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := model.NewRouter("chat", "tool")
			ctx := &msg.Context{}
			if tt.name == "handles nil context safely" {
				router.ApplyToContext(nil, false)
				return
			}
			router.ApplyToContext(ctx, tt.afterToolExecution)
			assert.Equal(t, tt.wantModel, ctx.ModelName)
		})
	}
}

func TestApplyDefaultRouter(t *testing.T) {
	tests := []struct {
		name      string
		cfg       msg.Config
		wantHook  bool
		wantModel string
	}{
		{name: "injects router hook", cfg: msg.Config{ChatModel: "chat", ToolModel: "tool"}, wantHook: true, wantModel: "chat"},
		{name: "skips when hook already set", cfg: msg.Config{ChatModel: "chat", ToolModel: "tool", PrepareNextTurn: func(msg.TurnContext) (*msg.TurnUpdate, error) {
			return nil, nil
		}}, wantHook: true},
		{name: "skips without dual models", cfg: msg.Config{ChatModel: "chat"}, wantHook: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := model.ApplyDefaultRouter(tt.cfg)
			if tt.wantHook {
				require.NotNil(t, got.PrepareNextTurn)
			} else {
				assert.Nil(t, got.PrepareNextTurn)
			}
			if tt.wantModel != "" {
				assert.Equal(t, tt.wantModel, got.ModelName)
			}
		})
	}
}

func TestRouter_PrepareNextTurnHook(t *testing.T) {
	tests := []struct {
		name        string
		toolResults int
		wantModel   string
	}{
		{name: "routes to tool model after tools", toolResults: 1, wantModel: "tool-model"},
		{name: "routes to chat model without tools", toolResults: 0, wantModel: "chat-model"},
		{name: "routes to tool model with multiple results", toolResults: 2, wantModel: "tool-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := model.NewRouter("chat-model", "tool-model")
			hook := router.PrepareNextTurnHook()
			toolResults := make([]msg.ToolResultMessage, tt.toolResults)
			update, err := hook(msg.TurnContext{
				Context:     &msg.Context{ModelName: "chat-model"},
				ToolResults: toolResults,
			})
			require.NoError(t, err)
			require.NotNil(t, update)
			assert.Equal(t, tt.wantModel, update.ModelName)
			assert.Equal(t, tt.wantModel, update.Context.ModelName)
		})
	}
}

func TestRouter_PrepareNextTurnHook_media(t *testing.T) {
	t.Parallel()

	imagePart := msg.MediaPart{Kind: msg.MediaKindImage, FileID: "img-1", MIMEType: "image/png"}
	audioPart := msg.MediaPart{Kind: msg.MediaKindAudio, FileID: "aud-1", MIMEType: "audio/wav"}
	userWithMedia := msg.UserMessage{Parts: []msg.ContentPart{
		msg.TextPart{Text: "describe"},
		imagePart,
		audioPart,
	}}

	tests := []struct {
		name           string
		chatModel      string
		toolModel      string
		toolResults    int
		wantMediaKinds []msg.MediaKind
	}{
		{
			name:           "strips media for text-only tool model",
			chatModel:      "mimo-v2.5",
			toolModel:      "deepseek-v4-pro",
			toolResults:    1,
			wantMediaKinds: nil,
		},
		{
			name:           "keeps media for omni-modal tool model",
			chatModel:      "deepseek-v4-pro",
			toolModel:      "mimo-v2.5",
			toolResults:    1,
			wantMediaKinds: []msg.MediaKind{msg.MediaKindImage, msg.MediaKindAudio},
		},
		{
			name:           "keeps image only for vision tool model",
			chatModel:      "mimo-v2.5",
			toolModel:      "gpt-5.3-codex",
			toolResults:    1,
			wantMediaKinds: []msg.MediaKind{msg.MediaKindImage},
		},
		{
			name:           "does not filter media on chat turn",
			chatModel:      "mimo-v2.5",
			toolModel:      "deepseek-v4-pro",
			toolResults:    0,
			wantMediaKinds: []msg.MediaKind{msg.MediaKindImage, msg.MediaKindAudio},
		},
		{
			name:           "does not filter when chat and tool models match",
			chatModel:      "deepseek-v4-pro",
			toolModel:      "deepseek-v4-pro",
			toolResults:    1,
			wantMediaKinds: []msg.MediaKind{msg.MediaKindImage, msg.MediaKindAudio},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := model.NewRouter(tt.chatModel, tt.toolModel)
			hook := router.PrepareNextTurnHook()
			update, err := hook(msg.TurnContext{
				Context: &msg.Context{
					ModelName: tt.chatModel,
					Messages:  []msg.AgentMessage{userWithMedia},
				},
				ToolResults: make([]msg.ToolResultMessage, tt.toolResults),
			})
			require.NoError(t, err)
			require.NotNil(t, update)
			require.Len(t, update.Context.Messages, 1)
			user, ok := update.Context.Messages[0].(msg.UserMessage)
			require.True(t, ok)

			var gotKinds []msg.MediaKind
			for _, part := range user.Parts {
				if mp, ok := part.(msg.MediaPart); ok {
					gotKinds = append(gotKinds, mp.Kind)
				}
			}
			assert.Equal(t, tt.wantMediaKinds, gotKinds)
		})
	}
}
