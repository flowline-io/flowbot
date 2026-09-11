package ctxmgr_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompactAndReloadBeforeCompactHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hook        ctxmgr.BeforeCompactFn
		wantErr     error
		wantSummary string
	}{
		{
			name: "cancel skips summarization",
			hook: func(context.Context, ctxmgr.BeforeCompactEvent) (*ctxmgr.BeforeCompactOutcome, error) {
				return &ctxmgr.BeforeCompactOutcome{Cancel: true}, nil
			},
			wantErr: ctxmgr.ErrCompactionCancelled,
		},
		{
			name: "custom compaction skips LLM",
			hook: func(_ context.Context, event ctxmgr.BeforeCompactEvent) (*ctxmgr.BeforeCompactOutcome, error) {
				return &ctxmgr.BeforeCompactOutcome{
					Compaction: &ctxmgr.CompactionResult{
						Summary:          "hook summary",
						FirstKeptEntryID: event.Preparation.FirstKeptEntryID,
						TokensBefore:     event.Preparation.TokensBefore,
					},
				}, nil
			},
			wantSummary: "hook summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nLLM summary"})
			store := session.NewMemoryStorage()
			sess := session.New(store)
			ctx := context.Background()
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage(strings.Repeat("x ", 2000)),
			}))
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("kept"),
			}))

			mgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: 1000,
				Settings: ctxmgr.Settings{
					Enabled:          true,
					ReserveTokens:    100,
					KeepRecentTokens: 2,
				},
				BeforeCompact: tt.hook,
			})

			report, err := mgr.CompactAndReload(ctx, sess, nil, ctxmgr.CompactOpts{
				Force:  true,
				Reason: ctxmgr.CompactReasonManual,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, 0, model.Calls())
				return
			}
			require.NoError(t, err)
			assert.True(t, report.Summarized)
			assert.Equal(t, 0, model.Calls())

			entries, err := store.ListEntries(ctx)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(entries), 3)
			leaf := entries[len(entries)-1]
			require.Equal(t, session.EntryCompaction, leaf.Type)
			assert.Equal(t, tt.wantSummary, leaf.Summary)
		})
	}
}

func TestMoveToBeforeTreeHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hook    ctxmgr.BeforeTreeFn
		wantErr error
	}{
		{
			name: "cancel leaves leaf unchanged",
			hook: func(context.Context, ctxmgr.BeforeTreeEvent) (*ctxmgr.BeforeTreeOutcome, error) {
				return &ctxmgr.BeforeTreeOutcome{Cancel: true}, nil
			},
			wantErr: ctxmgr.ErrTreeNavigationCancelled,
		},
		{
			name: "custom summary skips LLM",
			hook: func(context.Context, ctxmgr.BeforeTreeEvent) (*ctxmgr.BeforeTreeOutcome, error) {
				return &ctxmgr.BeforeTreeOutcome{Summary: "tree hook summary"}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "llm branch summary"})
			store := session.NewMemoryStorage()
			sess := session.New(store)
			ctx := context.Background()
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("root"),
			}))
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("child"),
			}))

			mgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: 4096,
				Settings:      ctxmgr.Settings{Enabled: true},
				BeforeTree:    tt.hook,
			})

			err := mgr.MoveTo(ctx, sess, "1", "")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				branch, branchErr := sess.GetBranch(ctx, "")
				require.NoError(t, branchErr)
				assert.Equal(t, "2", branch[len(branch)-1].ID)
				assert.Equal(t, 0, model.Calls())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 0, model.Calls())
			branch, err := sess.GetBranch(ctx, "")
			require.NoError(t, err)
			require.NotEmpty(t, branch)
			assert.Equal(t, "1", branch[0].ID)
			assert.Contains(t, branch[len(branch)-1].ID, "summary")
		})
	}
}

func TestMoveToBeforeTreeHookCallerSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		hook            ctxmgr.BeforeTreeFn
		wantErr         error
		wantUserWants   bool
		wantLeafContain string
	}{
		{
			name: "caller summary sets UserWantsSummary false",
			hook: func(_ context.Context, event ctxmgr.BeforeTreeEvent) (*ctxmgr.BeforeTreeOutcome, error) {
				assert.False(t, event.UserWantsSummary)
				return nil, nil
			},
			wantUserWants:   false,
			wantLeafContain: "summary",
		},
		{
			name: "cancel with caller summary leaves leaf unchanged",
			hook: func(_ context.Context, event ctxmgr.BeforeTreeEvent) (*ctxmgr.BeforeTreeOutcome, error) {
				assert.False(t, event.UserWantsSummary)
				return &ctxmgr.BeforeTreeOutcome{Cancel: true}, nil
			},
			wantErr: ctxmgr.ErrTreeNavigationCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "should not run"})
			store := session.NewMemoryStorage()
			sess := session.New(store)
			ctx := context.Background()
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("root"),
			}))
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("child"),
			}))

			mgr := ctxmgr.New(ctxmgr.Options{
				Model:         model,
				ModelName:     "fake",
				ContextWindow: 4096,
				Settings:      ctxmgr.Settings{Enabled: true},
				BeforeTree:    tt.hook,
			})

			err := mgr.MoveTo(ctx, sess, "1", "preset summary")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				branch, branchErr := sess.GetBranch(ctx, "")
				require.NoError(t, branchErr)
				assert.Equal(t, "2", branch[len(branch)-1].ID)
				assert.Equal(t, 0, model.Calls())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 0, model.Calls())
			branch, err := sess.GetBranch(ctx, "")
			require.NoError(t, err)
			assert.Contains(t, branch[len(branch)-1].ID, tt.wantLeafContain)
		})
	}
}
