package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestBaseTemplate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		data        map[string]any
		wantMinLen  int
		wantMsgRole string
	}{
		{
			name:        "with content",
			data:        map[string]any{"content": "hello"},
			wantMinLen:  2,
			wantMsgRole: llm.UserRole,
		},
		{
			name:        "empty content",
			data:        map[string]any{"content": ""},
			wantMinLen:  1,
			wantMsgRole: llm.SystemRole,
		},
		{
			name:        "no data parameter",
			data:        map[string]any{},
			wantMinLen:  1,
			wantMsgRole: llm.SystemRole,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			messages, err := llm.BaseTemplate().Format(context.Background(), tt.data)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(messages), tt.wantMinLen)
			assert.Equal(t, llm.SystemRole, messages[0].Role)
		})
	}
}

func TestDefaultTemplate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		data            map[string]any
		wantHasHistory  bool
		wantMessageRole string
	}{
		{
			name:            "no history no content",
			data:            map[string]any{},
			wantHasHistory:  false,
			wantMessageRole: llm.SystemRole,
		},
		{
			name: "with history",
			data: map[string]any{
				"chat_history": []*llm.Message{
					{Role: llm.UserRole, Content: "previous"},
				},
			},
			wantHasHistory:  true,
			wantMessageRole: llm.SystemRole,
		},
		{
			name: "with content",
			data: map[string]any{
				"content": "hello",
			},
			wantHasHistory:  false,
			wantMessageRole: llm.SystemRole,
		},
		{
			name: "with content and history",
			data: map[string]any{
				"content": "hello",
				"chat_history": []*llm.Message{
					{Role: llm.UserRole, Content: "previous"},
				},
			},
			wantHasHistory:  true,
			wantMessageRole: llm.SystemRole,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			messages, err := llm.DefaultTemplate().Format(context.Background(), tt.data)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(messages), 1)
			assert.Equal(t, llm.SystemRole, messages[0].Role)
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
	assert.Equal(t, llm.SystemRole, messages[0].Role)
}
