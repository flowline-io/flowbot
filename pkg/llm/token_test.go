package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestCountToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
	}{
		{name: "normal english text", text: "Hello world"},
		{name: "empty string", text: ""},
		{name: "chinese text", text: "你好世界"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := llm.CountToken(tt.text)
			if tt.text == "" {
				assert.Equal(t, 0, got)
			} else {
				assert.Positive(t, got)
			}
		})
	}
}

func TestCountMessageTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []*llm.Message
	}{
		{
			name:     "single message",
			messages: []*llm.Message{{Role: llm.UserRole, Content: "hello"}},
		},
		{
			name: "multiple messages",
			messages: []*llm.Message{
				{Role: llm.SystemRole, Content: "system prompt"},
				{Role: llm.UserRole, Content: "user question"},
				{Role: llm.AssistantRole, Content: "assistant response"},
			},
		},
		{
			name: "message with name",
			messages: []*llm.Message{
				{Role: llm.UserRole, Content: "hello", Name: "Alice"},
			},
		},
		{
			name:     "empty messages slice",
			messages: []*llm.Message{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := llm.CountMessageTokens(tt.messages)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, got, 0)
		})
	}
}
