package llm_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestFakeModel_GenerateContent(t *testing.T) {
	tests := []struct {
		name      string
		scripts   []llm.ResponseScript
		wantText  string
		wantCalls int
	}{
		{
			name:      "single text response",
			scripts:   []llm.ResponseScript{{Content: "hello"}},
			wantText:  "hello",
			wantCalls: 1,
		},
		{
			name: "tool call response",
			scripts: []llm.ResponseScript{{ToolCalls: []llms.ToolCall{{
				ID:           "call-1",
				Type:         "function",
				FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
			}}}},
			wantText:  "",
			wantCalls: 1,
		},
		{
			name:      "default done when queue empty",
			scripts:   nil,
			wantText:  "done",
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := llm.NewFakeModel(tt.scripts...)
			resp, err := model.GenerateContent(context.Background(), nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Choices)
			if tt.wantText != "" {
				assert.Equal(t, tt.wantText, resp.Choices[0].Content)
			}
			assert.Equal(t, tt.wantCalls, model.Calls())
		})
	}
}

func TestFakeModel_Streaming(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "streams delta", content: "abc"},
		{name: "streams word", content: "hello"},
		{name: "streams empty", content: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := llm.NewFakeModel(llm.ResponseScript{Content: tt.content})
			var streamed string
			_, err := model.GenerateContent(context.Background(), nil, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				streamed += string(chunk)
				return nil
			}))
			require.NoError(t, err)
			assert.Equal(t, tt.content, streamed)
		})
	}
}

func TestFakeModel_MultiChunkStreaming(t *testing.T) {
	tests := []struct {
		name       string
		chunks     []string
		wantStream string
		wantText   string
	}{
		{name: "three chunks", chunks: []string{"hel", "lo", " world"}, wantStream: "hello world", wantText: "hello world"},
		{name: "single chunk slice", chunks: []string{"only"}, wantStream: "only", wantText: "only"},
		{name: "content overrides join", chunks: []string{"a", "b"}, wantStream: "ab", wantText: "explicit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			script := llm.ResponseScript{Chunks: tt.chunks}
			if tt.name == "content overrides join" {
				script.Content = "explicit"
			}
			model := llm.NewFakeModel(script)
			var streamed string
			resp, err := model.GenerateContent(context.Background(), nil, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				streamed += string(chunk)
				return nil
			}))
			require.NoError(t, err)
			require.NotEmpty(t, resp.Choices)
			if tt.name == "content overrides join" {
				assert.Equal(t, "ab", streamed)
			} else {
				assert.Equal(t, tt.wantStream, streamed)
			}
			assert.Equal(t, tt.wantText, resp.Choices[0].Content)
		})
	}
}

func TestFakeModel_ContextCancelled(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancel before call"},
		{name: "cancel stops call"},
		{name: "cancel returns error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			model := llm.NewFakeModel(llm.ResponseScript{Content: "x"})
			_, err := model.GenerateContent(ctx, nil)
			assert.Error(t, err)
		})
	}
}
