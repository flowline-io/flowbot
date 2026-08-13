package ctxmgr_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
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
					ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage(long),
				}); err != nil {
					return err
				}
				return s.Append(ctx, session.TreeEntry{
					ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("recent"),
				})
			},
			contextWindow:  1000,
			wantCompaction: true,
		},
		{
			name: "skips short history",
			setup: func(ctx context.Context, s *session.Session) error {
				return s.Append(ctx, session.TreeEntry{
					ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("hi"),
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

			ag := loop.NewAgent(loop.Options{
				InitialState: &msg.Context{SystemPrompt: "system"},
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

func TestManagerEnsureWithinBudgetPruneSkipsSummary(t *testing.T) {
	t.Parallel()

	appendToolTurn := func(ctx context.Context, t *testing.T, sess *session.Session, toolText string) {
		t.Helper()
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("read the file"),
		}))
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.AssistantMessage{
				Parts: []msg.ContentPart{msg.ToolCallPart{ID: "c1", Name: "read", Arguments: `{}`}},
			},
		}))
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "3", ParentID: "2", Type: session.EntryMessage, Message: msg.ToolResultMessage{
				ToolCallID: "c1",
				Name:       "read",
				Parts:      []msg.ContentPart{msg.TextPart{Text: toolText}},
			},
		}))
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "4", ParentID: "3", Type: session.EntryMessage, Message: msg.NewUserMessage("continue"),
		}))
	}

	tests := []struct {
		name           string
		prune          bool
		toolText       string
		contextWindow  int
		wantCalls      int
		wantPruned     bool
		wantCompaction bool
	}{
		{
			name:           "pruned tool result skips summarization",
			prune:          true,
			toolText:       strings.Repeat("a", 20000),
			contextWindow:  4000,
			wantCalls:      0,
			wantPruned:     true,
			wantCompaction: false,
		},
		{
			name:           "prune disabled still summarizes when over threshold",
			prune:          false,
			toolText:       strings.Repeat("a", 20000),
			contextWindow:  4000,
			wantCalls:      1,
			wantPruned:     false,
			wantCompaction: true,
		},
		{
			name:           "already under budget skips prune and summary",
			prune:          true,
			toolText:       "small",
			contextWindow:  128000,
			wantCalls:      0,
			wantPruned:     false,
			wantCompaction: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := session.NewMemoryStorage()
			sess := session.New(store)
			appendToolTurn(ctx, t, sess, tt.toolText)

			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nshould not run"})
			mgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: tt.contextWindow,
				Settings:      ctxmgr.Settings{Enabled: true, PruneToolOutputs: tt.prune, ReserveTokens: 500, KeepRecentTokens: 2},
				SystemPrompt:  "system",
			})
			ag := loop.NewAgent(loop.Options{
				InitialState: &msg.Context{SystemPrompt: "system"},
			})
			require.NoError(t, mgr.EnsureWithinBudget(ctx, sess, ag))
			assert.Equal(t, tt.wantCalls, model.Calls())

			entries, err := store.ListEntries(ctx)
			require.NoError(t, err)
			hasCompaction := false
			foundPruned := false
			for _, entry := range entries {
				if entry.Type == session.EntryCompaction {
					hasCompaction = true
				}
				toolResult, ok := entry.Message.(msg.ToolResultMessage)
				if !ok {
					continue
				}
				for _, part := range toolResult.Parts {
					if tp, ok := part.(msg.TextPart); ok && strings.Contains(tp.Text, "[... tool result middle pruned ...]") {
						foundPruned = true
					}
				}
			}
			assert.Equal(t, tt.wantCompaction, hasCompaction)
			assert.Equal(t, tt.wantPruned, foundPruned)
		})
	}
}
