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
