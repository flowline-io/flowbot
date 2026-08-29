package llm_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestIsRetryableLLMError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "rate limit", err: errors.New("429 rate limit exceeded"), want: true},
		{name: "timeout", err: errors.New("request timeout"), want: true},
		{name: "langchaingo sanitized network error", err: errors.New("network error: failed to reach API server"), want: true},
		{name: "503", err: errors.New("503 service unavailable"), want: true},
		{name: "overflow", err: errors.New("exceeds the context window"), want: false},
		{name: "auth", err: errors.New("unauthorized invalid api key"), want: false},
		{name: "aborted", err: agentllm.ErrAborted, want: false},
		{name: "stream started", err: agentllm.ErrStreamStarted, want: false},
		{name: "stream idle", err: agentllm.ErrStreamIdle, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, agentllm.IsRetryableLLMError(tt.err))
		})
	}
}

func TestStreamAssistantRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		scripts   []agentllm.ResponseScript
		wantCalls int
		wantErr   error
		wantSub   string
		wantText  string
	}{
		{
			name: "retries then succeeds",
			scripts: []agentllm.ResponseScript{
				{Err: errors.New("429 rate limit")},
				{Content: "ok"},
			},
			wantCalls: 2,
			wantText:  "ok",
		},
		{
			name: "retries langchaingo sanitized network error",
			scripts: []agentllm.ResponseScript{
				{Err: errors.New("network error: failed to reach API server")},
				{Content: "ok"},
			},
			wantCalls: 2,
			wantText:  "ok",
		},
		{
			name: "non retryable fails immediately",
			scripts: []agentllm.ResponseScript{
				{Err: errors.New("unauthorized invalid api key")},
				{Content: "ok"},
			},
			wantCalls: 1,
			wantSub:   "unauthorized",
		},
		{
			name: "no retry after stream started",
			scripts: []agentllm.ResponseScript{
				{Chunks: []string{"hi"}, ErrAfterStream: errors.New("429 rate limit")},
				{Content: "ok"},
			},
			wantCalls: 1,
			wantErr:   agentllm.ErrStreamStarted,
		},
		{
			name: "no retry after stream idle once started",
			scripts: []agentllm.ResponseScript{
				{Chunks: []string{"hi"}, ErrAfterStream: agentllm.ErrStreamIdle},
				{Content: "ok"},
			},
			wantCalls: 1,
			wantErr:   agentllm.ErrStreamStarted,
		},
		{
			name: "retries stream idle before any delta",
			scripts: []agentllm.ResponseScript{
				{Err: agentllm.ErrStreamIdle},
				{Content: "recovered"},
			},
			wantCalls: 2,
			wantText:  "recovered",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(tt.scripts...)
			result, err := agentllm.StreamAssistant(context.Background(), model, "", nil, agentllm.StreamOptions{
				ModelName: "test",
				Retry: agentllm.RetryConfig{
					MaxAttempts:     3,
					InitialInterval: time.Millisecond,
					MaxInterval:     5 * time.Millisecond,
					Multiplier:      1,
				},
				OnTextDelta: func(string) error { return nil },
			})
			assert.Equal(t, tt.wantCalls, model.Calls())
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.wantSub != "" {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), tt.wantSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, result.Content)
		})
	}
}

func TestStreamAssistantAbort(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
	_, err := agentllm.StreamAssistant(ctx, model, "", nil, agentllm.StreamOptions{ModelName: "test"})
	require.ErrorIs(t, err, agentllm.ErrAborted)
	assert.Equal(t, 0, model.Calls())
}

func TestCompleteWithRetry(t *testing.T) {
	t.Parallel()
	model := agentllm.NewFakeModel(
		agentllm.ResponseScript{Err: errors.New("502 bad gateway")},
		agentllm.ResponseScript{Content: "summary"},
	)
	got, err := agentllm.CompleteWithRetry(
		context.Background(),
		model,
		"",
		[]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "hi")},
		"test",
		0,
		agentllm.RetryConfig{MaxAttempts: 3, InitialInterval: time.Millisecond, MaxInterval: 5 * time.Millisecond, Multiplier: 1},
	)
	require.NoError(t, err)
	assert.Equal(t, "summary", got)
	assert.Equal(t, 2, model.Calls())
}

func TestCompleteWithTools(t *testing.T) {
	t.Parallel()
	echoTool := llms.Tool{
		Type:     "function",
		Function: &llms.FunctionDefinition{Name: "echo", Description: "echo"},
	}
	tests := []struct {
		name          string
		tools         []llms.Tool
		thinkingLevel string
		reasoning     map[string]string
		wantName      string
	}{
		{
			name:          "forwards tools thinking and reasoning on the completion call",
			tools:         []llms.Tool{echoTool},
			thinkingLevel: agentllm.ThinkingLevelHigh,
			reasoning:     map[string]string{"c1": "prior think"},
			wantName:      "echo",
		},
		{name: "forwards empty tools when the list is nil", tools: nil},
		{name: "forwards empty tools when the list has no entries", tools: []llms.Tool{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			got, err := agentllm.CompleteWithTools(context.Background(), agentllm.CompleteRequest{
				Model:                  model,
				SystemPrompt:           "system",
				Messages:               []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "hi")},
				ModelName:              "test",
				Tools:                  tt.tools,
				ThinkingLevel:          tt.thinkingLevel,
				AssistantToolReasoning: tt.reasoning,
			})
			require.NoError(t, err)
			assert.Equal(t, "ok", got)
			if tt.wantName == "" {
				assert.Empty(t, model.LastTools())
			} else {
				require.NotEmpty(t, model.LastTools())
				assert.Equal(t, tt.wantName, model.LastTools()[0].Function.Name)
			}
			wantLevel := agentllm.ThinkingLevelDefault
			if tt.thinkingLevel != "" {
				wantLevel = tt.thinkingLevel
			}
			assert.Equal(t, wantLevel, agentllm.ThinkingLevelFromContext(model.LastContext()))
			assert.Equal(t, tt.reasoning, agentllm.AssistantToolReasoningFromContext(model.LastContext()))
		})
	}
}

func TestRetryConfigFromChatAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  config.LLMRetryConfig
		want int
	}{
		{name: "default", cfg: config.LLMRetryConfig{}, want: 3},
		{name: "custom", cfg: config.LLMRetryConfig{MaxAttempts: 5}, want: 5},
		{name: "one", cfg: config.LLMRetryConfig{MaxAttempts: 1}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := agentllm.RetryConfigFromChatAgent(tt.cfg)
			assert.Equal(t, tt.want, got.MaxAttempts)
		})
	}
}
