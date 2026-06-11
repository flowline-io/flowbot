package harness_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:    "single success",
			scripts: []agentllm.ResponseScript{{Content: "ok"}},
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

			model := agentllm.NewFakeModel(tt.scripts...)
			ctxMgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: 128000,
				Settings:      ctxmgr.Settings{Enabled: true, ReserveTokens: 16384, KeepRecentTokens: 2},
				SystemPrompt:  "system",
			})
			h := harness.New(harness.Options{
				AgentOptions:   agent.Options{Model: model},
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
	model := agentllm.NewFakeModel(agentllm.ResponseScript{
		Err: fmt.Errorf("Your input exceeds the context window of this model"),
	})
	ctxMgr := ctxmgr.New(ctxmgr.Options{
		Model:         model,
		ModelName:     "fake",
		ContextWindow: 128000,
		Settings:      ctxmgr.Settings{Enabled: false},
	})
	h := harness.New(harness.Options{
		AgentOptions:   agent.Options{Model: model},
		ContextManager: ctxMgr,
		ModelName:      "fake",
	})
	_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, h.WaitIdle(ctx))
	require.Error(t, h.LastRunResult().Err)
	assert.Equal(t, 1, model.Calls())
}
