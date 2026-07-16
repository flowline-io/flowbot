package ctxmgr_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerSettingsAndPrompt(t *testing.T) {
	t.Parallel()

	settings := ctxmgr.Settings{Enabled: true, ReserveTokens: 500, KeepRecentTokens: 1000}
	mgr := ctxmgr.New(ctxmgr.Options{
		Model:         agentllm.NewFakeModel(agentllm.ResponseScript{Content: "summary"}),
		ModelName:     "fake",
		ContextWindow: 4096,
		Settings:      settings,
		SystemPrompt:  "system",
	})

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "settings snapshot",
			run: func(t *testing.T) {
				got := mgr.Settings()
				assert.True(t, got.Enabled)
				assert.Equal(t, 500, got.ReserveTokens)
			},
		},
		{
			name: "context window",
			run: func(t *testing.T) {
				assert.Equal(t, 4096, mgr.ContextWindow())
			},
		},
		{
			name: "update system prompt affects usage",
			run: func(t *testing.T) {
				mgr.UpdateSystemPrompt("longer system prompt for usage")
				usage := mgr.GetContextUsage(nil)
				assert.Positive(t, usage.Tokens)
				assert.Equal(t, 4096, usage.ContextWindow)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestManagerMoveTo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nMoved branch"})

	tests := []struct {
		name        string
		setup       func(*session.Session) (oldLeaf, target string)
		withSummary bool
		wantErr     bool
	}{
		{
			name: "explicit summary skips generation",
			setup: func(s *session.Session) (string, string) {
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("root")}))
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "leaf", ParentID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("leaf")}))
				return "leaf", "root"
			},
			withSummary: true,
		},
		{
			name: "same leaf is no-op",
			setup: func(s *session.Session) (string, string) {
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "only", Type: session.EntryMessage, Message: agent.NewUserMessage("only")}))
				return "only", "only"
			},
		},
		{
			name: "branch switch summarizes abandoned path",
			setup: func(s *session.Session) (string, string) {
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("root")}))
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "left", ParentID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("left")}))
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "old", ParentID: "left", Type: session.EntryMessage, Message: agent.NewUserMessage("old work")}))
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "right", ParentID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("right")}))
				return "old", "right"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := session.NewMemoryStorage()
			sess := session.New(store)
			_, target := tt.setup(sess)

			mgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: 128000,
				Settings:      ctxmgr.Settings{Enabled: true, KeepRecentTokens: 100000},
			})

			summary := ""
			if tt.withSummary {
				summary = "preset summary"
			}
			err := mgr.MoveTo(ctx, sess, target, summary)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestManagerReloadAgentState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := session.NewMemoryStorage()
	sess := session.New(store)
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("stored"),
	}))

	mgr := ctxmgr.New(ctxmgr.Options{
		Model:         agentllm.NewFakeModel(),
		ModelName:     "fake",
		ContextWindow: 4096,
		SystemPrompt:  "system",
	})
	ag := agent.NewAgent(agent.Options{
		InitialState: &agent.Context{Messages: []agent.AgentMessage{agent.NewUserMessage("extra")}},
	})

	require.NoError(t, mgr.ReloadAgentState(ctx, sess, ag))
	state := ag.State()
	require.NotEmpty(t, state.Messages)
	assert.Equal(t, "system", state.SystemPrompt)
}
