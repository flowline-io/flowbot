package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateWithModel_fakeModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scripts []ResponseScript
		prompt  string
		want    string
	}{
		{
			name:    "returns scripted content",
			scripts: []ResponseScript{{Content: "tag-one, tag-two"}},
			prompt:  "extract tags",
			want:    "tag-one, tag-two",
		},
		{
			name:    "empty prompt still completes",
			scripts: []ResponseScript{{Content: "done"}},
			prompt:  "",
			want:    "done",
		},
		{
			name:    "propagates model error",
			scripts: []ResponseScript{{Err: errors.New("model failed")}},
			prompt:  "fail",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model := NewFakeModel(tt.scripts...)
			messages, err := BaseTemplate().Format(context.Background(), map[string]any{
				"content": tt.prompt,
			})
			require.NoError(t, err)

			got, err := generateWithModel(context.Background(), model, "fake-model", messages)
			if tt.scripts[0].Err != nil {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLLMGenerate_emptyModel(t *testing.T) {
	t.Parallel()

	_, err := LLMGenerate(context.Background(), "", "hello")
	assert.Error(t, err)
}
