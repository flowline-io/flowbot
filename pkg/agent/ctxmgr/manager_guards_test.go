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

func TestManagerNilSessionGuards(t *testing.T) {
	t.Parallel()

	mgr := ctxmgr.New(ctxmgr.Options{
		Model:         agentllm.NewFakeModel(agentllm.ResponseScript{Content: "summary"}),
		ModelName:     "fake",
		ContextWindow: 4096,
		Settings:      ctxmgr.Settings{Enabled: true},
	})

	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{
			name: "ensure within budget nil session",
			run: func() error {
				return mgr.EnsureWithinBudget(context.Background(), nil, nil)
			},
		},
		{
			name:    "compact and reload nil session",
			run:     func() error { return mgr.CompactAndReload(context.Background(), nil, nil, ctxmgr.CompactOpts{}) },
			wantErr: true,
		},
		{
			name:    "move to nil session",
			run:     func() error { return mgr.MoveTo(context.Background(), nil, "x", "") },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.run()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRunCompaction(t *testing.T) {
	t.Parallel()

	model := agentllm.NewFakeModel(
		agentllm.ResponseScript{Content: "## Goal\nCompacted"},
		agentllm.ResponseScript{Content: "prefix chunk"},
	)

	tests := []struct {
		name     string
		prep     *ctxmgr.CompactionPreparation
		wantOK   bool
		wantText string
	}{
		{
			name: "split turn compacts prefix and history",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID:    "keep",
				MessagesToSummarize: []agent.AgentMessage{agent.NewUserMessage("history")},
				TurnPrefixMessages:  []agent.AgentMessage{agent.NewUserMessage("prefix")},
				IsSplitTurn:         true,
				FileOps:             ctxmgr.NewFileOperations(),
				Settings:            ctxmgr.Settings{},
			},
			wantOK:   true,
			wantText: "Compacted",
		},
		{
			name: "empty messages returns nothing to compact",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID: "keep",
				FileOps:          ctxmgr.NewFileOperations(),
				Settings:         ctxmgr.Settings{},
			},
			wantOK: false,
		},
		{
			name:   "nil preparation returns error result",
			prep:   nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.prep == nil {
				got := ctxmgr.RunCompaction(context.Background(), model, "fake", nil)
				assert.False(t, got.IsOk())
				return
			}
			got := ctxmgr.RunCompaction(context.Background(), model, "fake", tt.prep)
			require.Equal(t, tt.wantOK, got.IsOk())
			if tt.wantText != "" {
				assert.Contains(t, got.Value().Summary, tt.wantText)
			}
		})
	}
}

func TestManagerCompactAndReloadDisabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := session.NewMemoryStorage()
	sess := session.New(store)
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("hello"),
	}))

	mgr := ctxmgr.New(ctxmgr.Options{
		Model:         agentllm.NewFakeModel(),
		ModelName:     "fake",
		ContextWindow: 4096,
		Settings:      ctxmgr.Settings{Enabled: false},
	})

	require.NoError(t, mgr.CompactAndReload(ctx, sess, nil, ctxmgr.CompactOpts{Force: false}))
}
