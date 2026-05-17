package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestChatModel_OpenAI(t *testing.T) {
	t.Parallel()

	client, err := llm.ChatModel(context.Background(), "gpt-5.5-instant")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestChatModel_Gemini(t *testing.T) {
	t.Parallel()

	client, err := llm.ChatModel(context.Background(), "gemini-3.1-pro")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestChatModel_Anthropic(t *testing.T) {
	t.Parallel()

	client, err := llm.ChatModel(context.Background(), "claude-opus-4.7")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestChatModel_EmptyModelName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
	}{
		{name: "empty string", modelName: ""},
		{name: "unknown model", modelName: "nonexistent-model"},
		{name: "whitespace-only model name", modelName: "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := llm.ChatModel(context.Background(), tt.modelName)
			require.Error(t, err)
			assert.Nil(t, client)
		})
	}
}

func TestGetModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelName string
		wantProv  string
		wantOK    bool
	}{
		{name: "known openai model", modelName: "gpt-5.5-instant", wantProv: llm.ProviderOpenAI, wantOK: true},
		{name: "known gemini model", modelName: "gemini-3.1-pro", wantProv: llm.ProviderGemini, wantOK: true},
		{name: "unknown model", modelName: "unknown-model", wantProv: "", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := llm.GetModel(tt.modelName)
			if tt.wantOK {
				assert.Equal(t, tt.wantProv, m.Provider)
			} else {
				assert.Empty(t, m.Provider)
			}
		})
	}
}
