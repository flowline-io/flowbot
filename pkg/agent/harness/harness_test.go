package harness_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestHarnessOverflowRetryUsesFinalResult(t *testing.T) {
	tests := []struct {
		name       string
		scripts    []agentllm.ResponseScript
		wantErr    bool
		wantSubstr string
	}{
		{
			name: "overflow then success",
			scripts: []agentllm.ResponseScript{
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
				{Content: "## Goal\nCompact summary"},
				{Content: "recovered reply"},
			},
			wantSubstr: "recovered reply",
		},
		{
			name:       "single success",
			scripts:    []agentllm.ResponseScript{{Content: "ok"}},
			wantSubstr: "ok",
		},
		{
			name: "overflow without recovery script",
			scripts: []agentllm.ResponseScript{
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
				{Content: "## Goal\nCompact summary"},
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := session.NewMemoryStorage()
			sess := session.New(store)
			long := strings.Repeat("word ", 5000)
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage(long),
			}))
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("recent"),
			}))

			fakeModel := agentllm.NewFakeModel(tt.scripts...)
			ctxMgr := ctxmgr.New(ctxmgr.Options{
				Model:         fakeModel,
				ModelName:     "fake",
				ContextWindow: 128000,
				Settings:      ctxmgr.Settings{Enabled: true, ReserveTokens: 16384, KeepRecentTokens: 2},
				SystemPrompt:  "system",
			})
			h := harness.New(harness.Options{
				AgentOptions:   agent.Options{Model: fakeModel},
				Session:        sess,
				ContextManager: ctxMgr,
				SystemPrompt:   "system",
				ModelName:      "fake",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))

			result := h.LastRunResult()
			if tt.wantErr {
				assert.Error(t, result.Err)
				return
			}
			require.NoError(t, result.Err)
			reply := ""
			for i := len(result.Messages) - 1; i >= 0; i-- {
				assistant, ok := result.Messages[i].(agent.AssistantMessage)
				if !ok {
					continue
				}
				reply = assistant.TextContent()
				if reply != "" {
					break
				}
			}
			assert.Contains(t, reply, tt.wantSubstr)
		})
	}
}

func TestHarnessRespectsCompactionDisabledOnOverflow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{
		Err: fmt.Errorf("Your input exceeds the context window of this model"),
	})
	ctxMgr := ctxmgr.New(ctxmgr.Options{
		Model:         fakeModel,
		ModelName:     "fake",
		ContextWindow: 128000,
		Settings:      ctxmgr.Settings{Enabled: false},
	})
	h := harness.New(harness.Options{
		AgentOptions:   agent.Options{Model: fakeModel},
		ContextManager: ctxMgr,
		ModelName:      "fake",
	})
	_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, h.WaitIdle(ctx))
	require.Error(t, h.LastRunResult().Err)
	assert.Equal(t, 1, fakeModel.Calls())
}

func TestHarnessRouterDualModelRouting(t *testing.T) {
	tests := []struct {
		name      string
		scripts   []agentllm.ResponseScript
		wantModel []string
	}{
		{
			name: "router sync routes after tool execution",
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
				}}},
				{Content: "done"},
			},
			wantModel: []string{"chat-model", "tool-model"},
		},
		{
			name: "router keeps tool model on chained tool rounds",
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"one"}`},
				}}},
				{ToolCalls: []llms.ToolCall{{
					ID: "call-2", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"two"}`},
				}}},
				{Content: "done"},
			},
			wantModel: []string{"chat-model", "tool-model", "tool-model"},
		},
		{
			name:      "router without tools stays on chat model",
			scripts:   []agentllm.ResponseScript{{Content: "ok"}},
			wantModel: []string{"chat-model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fakeModel := agentllm.NewFakeModel(tt.scripts...)
			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(echo.Tool{}))

			h := harness.New(harness.Options{
				AgentOptions: agent.Options{
					Model:    fakeModel,
					Registry: reg,
					Config:   agent.Config{MaxSteps: 10},
				},
				Router:    model.NewRouter("chat-model", "tool-model"),
				ModelName: "chat-model",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("run"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			require.NoError(t, h.LastRunResult().Err)

			got := make([]string, 0)
			for _, item := range h.LastRunResult().Messages {
				assistant, ok := item.(agent.AssistantMessage)
				if !ok {
					continue
				}
				got = append(got, assistant.Model)
			}
			assert.Equal(t, tt.wantModel, got)
		})
	}
}
