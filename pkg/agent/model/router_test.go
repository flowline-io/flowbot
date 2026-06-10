package model_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
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
		name string
	}{
		{name: "updates model on context"},
		{name: "handles nil context"},
		{name: "handles nil router"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := model.NewRouter("chat", "tool")
			ctx := &msg.Context{}
			router.ApplyToContext(ctx, true)
			assert.Equal(t, "tool", ctx.ModelName)
			router.ApplyToContext(nil, false)
		})
	}
}
