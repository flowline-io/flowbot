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

func TestPrepareCompactionForceAfterCompactionLeaf(t *testing.T) {
	base := []session.TreeEntry{
		{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("old")},
		{ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("kept")},
		{ID: "3", ParentID: "2", Type: session.EntryCompaction, Summary: "summary", FirstKeptEntryID: "2"},
	}

	tests := []struct {
		name      string
		force     bool
		extra     []agent.AgentMessage
		wantNil   bool
		wantFirst string
	}{
		{name: "force without extra", force: true, wantFirst: "2"},
		{name: "force with extra messages", force: true, extra: []agent.AgentMessage{agent.NewUserMessage("pending turn")}, wantFirst: "2"},
		{name: "no force returns nil", force: false, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := ctxmgr.PrepareOptions{Force: tt.force, ExtraMessages: tt.extra}
			gotResult := ctxmgr.PrepareCompaction(base, ctxmgr.Settings{Enabled: true, KeepRecentTokens: 100000}, opts)
			require.True(t, gotResult.IsOk())
			got := gotResult.Value()
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantFirst, got.FirstKeptEntryID)
			assert.NotEmpty(t, got.MessagesToSummarize)
		})
	}
}

func TestCompactAndReloadForceWithExtraMessages(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := session.NewMemoryStorage()
	sess := session.New(store)
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage(strings.Repeat("x ", 2000)),
	}))
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("kept"),
	}))
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "3", ParentID: "2", Type: session.EntryCompaction, Summary: "old summary", FirstKeptEntryID: "2",
	}))

	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nUpdated"})
	mgr := ctxmgr.New(ctxmgr.Options{
		Model: model, ModelName: "fake", ContextWindow: 1000,
		Settings:     ctxmgr.Settings{Enabled: true, ReserveTokens: 100, KeepRecentTokens: 2},
		SystemPrompt: "system",
	})
	ag := agent.NewAgent(agent.Options{
		InitialState: &agent.Context{
			SystemPrompt: "system",
			Messages: append(session.BuildContext([]session.TreeEntry{
				{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage(strings.Repeat("x ", 2000))},
				{ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("kept")},
				{ID: "3", ParentID: "2", Type: session.EntryCompaction, Summary: "old summary", FirstKeptEntryID: "2"},
			}).Messages, agent.NewUserMessage("unsaved user turn")),
		},
	})

	err := mgr.CompactAndReload(ctx, sess, ag, ctxmgr.CompactOpts{Force: true})
	require.NoError(t, err)

	entries, err := store.ListEntries(ctx)
	require.NoError(t, err)
	assert.Len(t, entries, 4)
	assert.Equal(t, session.EntryCompaction, entries[3].Type)
}

func TestIsContextOverflowMessage(t *testing.T) {
	tests := []struct {
		name    string
		message agent.AssistantMessage
		window  int
		want    bool
	}{
		{name: "error text", message: agent.AssistantMessage{StopReason: "error", Parts: []agent.ContentPart{
			agent.TextPart{Text: "prompt is too long"},
		}}, window: 128000, want: true},
		{name: "silent usage overflow", message: agent.AssistantMessage{
			StopReason: "complete",
			Usage:      &agent.Usage{PromptTokens: 130000},
		}, window: 128000, want: true},
		{name: "normal", message: agent.AssistantMessage{
			StopReason: "complete",
			Usage:      &agent.Usage{PromptTokens: 1000},
		}, window: 128000, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ctxmgr.IsContextOverflowMessage(tt.message, tt.window))
		})
	}
}
