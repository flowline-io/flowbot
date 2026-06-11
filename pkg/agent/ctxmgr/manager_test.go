package ctxmgr_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerEnsureWithinBudgetCompacts(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(context.Context, *session.Session) error
		contextWindow  int
		wantCompaction bool
	}{
		{
			name: "compacts long history",
			setup: func(ctx context.Context, s *session.Session) error {
				long := strings.Repeat("word ", 5000)
				if err := s.Append(ctx, session.TreeEntry{
					ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage(long),
				}); err != nil {
					return err
				}
				return s.Append(ctx, session.TreeEntry{
					ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("recent"),
				})
			},
			contextWindow:  1000,
			wantCompaction: true,
		},
		{
			name: "skips short history",
			setup: func(ctx context.Context, s *session.Session) error {
				return s.Append(ctx, session.TreeEntry{
					ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("hi"),
				})
			},
			contextWindow:  128000,
			wantCompaction: false,
		},
		{
			name:           "empty session",
			setup:          func(context.Context, *session.Session) error { return nil },
			contextWindow:  1000,
			wantCompaction: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := session.NewMemoryStorage()
			sess := session.New(store)
			require.NoError(t, tt.setup(ctx, sess))

			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nTest summary"})
			mgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: tt.contextWindow,
				Settings:      ctxmgr.Settings{Enabled: true, ReserveTokens: 100, KeepRecentTokens: 2},
				SystemPrompt:  "system",
			})

			ag := agent.NewAgent(agent.Options{
				InitialState: &agent.Context{SystemPrompt: "system"},
			})
			err := mgr.EnsureWithinBudget(ctx, sess, ag)
			require.NoError(t, err)

			entries, err := store.ListEntries(ctx)
			require.NoError(t, err)
			hasCompaction := false
			for _, entry := range entries {
				if entry.Type == session.EntryCompaction {
					hasCompaction = true
				}
			}
			assert.Equal(t, tt.wantCompaction, hasCompaction)
		})
	}
}
