package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
)

func TestBaseTemplate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		data       map[string]any
		wantMinLen int
	}{
		{
			name:       "with content",
			data:       map[string]any{"content": "hello"},
			wantMinLen: 2,
		},
		{
			name:       "empty content",
			data:       map[string]any{"content": ""},
			wantMinLen: 1,
		},
		{
			name:       "no data parameter",
			data:       map[string]any{},
			wantMinLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			messages, err := llm.BaseTemplate().Format(context.Background(), tt.data)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(messages), tt.wantMinLen)
			assert.Equal(t, llms.ChatMessageTypeSystem, messages[0].Role)
		})
	}
}

func TestDefaultTemplate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		data       map[string]any
		wantMinLen int
		wantErr    bool
	}{
		{
			name:       "no history no content",
			data:       map[string]any{},
			wantMinLen: 1,
		},
		{
			name: "with history",
			data: map[string]any{
				"chat_history": []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeHuman, "previous"),
				},
			},
			wantMinLen: 2,
		},
		{
			name: "with content",
			data: map[string]any{
				"content": "hello",
			},
			wantMinLen: 2,
		},
		{
			name: "with content and history",
			data: map[string]any{
				"content": "hello",
				"chat_history": []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeHuman, "previous"),
				},
			},
			wantMinLen: 3,
		},
		{
			name: "invalid chat history type",
			data: map[string]any{
				"chat_history": "not-a-slice",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			messages, err := llm.DefaultTemplate().Format(context.Background(), tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(messages), tt.wantMinLen)
			assert.Equal(t, llms.ChatMessageTypeSystem, messages[0].Role)
		})
	}
}

func TestDefaultMultiChatTemplate_Format(t *testing.T) {
	t.Parallel()

	messages, err := llm.DefaultMultiChatTemplate().Format(context.Background(), map[string]any{
		"content": "test",
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(messages), 2)
	assert.Equal(t, llms.ChatMessageTypeSystem, messages[0].Role)
}

func TestNewModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		wantErr   bool
	}{
		{name: "known openai model", modelName: "gpt-5.5-instant", wantErr: false},
		{name: "known gemini model", modelName: "gemini-3.1-pro", wantErr: false},
		{name: "known anthropic model", modelName: "claude-opus-4.7", wantErr: false},
		{name: "anthropic model with base url", modelName: "claude-proxy", wantErr: false},
		{name: "unknown model", modelName: "unknown-model", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := llm.NewModel(context.Background(), tt.modelName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestGenerateWithTemplate_emptyModel(t *testing.T) {
	t.Parallel()

	_, err := llm.GenerateWithTemplate(context.Background(), "", llm.BaseTemplate(), map[string]any{
		"content": "hello",
	})
	assert.Error(t, err)
}
