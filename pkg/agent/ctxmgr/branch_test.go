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

func branchTree() []session.TreeEntry {
	return []session.TreeEntry{
		{ID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("root")},
		{ID: "left", ParentID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("left")},
		{ID: "old-leaf", ParentID: "left", Type: session.EntryMessage, Message: agent.NewUserMessage("old")},
		{ID: "right", ParentID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("right")},
		{ID: "new-leaf", ParentID: "right", Type: session.EntryMessage, Message: agent.NewUserMessage("new")},
	}
}

func TestCollectBranchEntries(t *testing.T) {
	t.Parallel()

	entries := branchTree()

	tests := []struct {
		name       string
		oldLeaf    string
		newEntry   string
		wantOK     bool
		wantCount  int
		wantCommon string
	}{
		{name: "shared root yields no prefix entries", oldLeaf: "old-leaf", newEntry: "new-leaf", wantOK: true, wantCount: 0, wantCommon: "root"},
		{name: "same leaf collects prefix before leaf", oldLeaf: "old-leaf", newEntry: "old-leaf", wantOK: true, wantCount: 2, wantCommon: "old-leaf"},
		{name: "empty id returns error", oldLeaf: "", newEntry: "new-leaf", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.CollectBranchEntries(entries, tt.oldLeaf, tt.newEntry)
			if !tt.wantOK {
				assert.False(t, got.IsOk())
				return
			}
			require.True(t, got.IsOk())
			value := got.Value()
			assert.Len(t, value.Entries, tt.wantCount)
			if tt.wantCommon != "" {
				assert.Equal(t, tt.wantCommon, value.CommonAncestor)
			}
		})
	}
}

func TestPrepareBranchSummary(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("word ", 200)
	entries := []session.TreeEntry{
		{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage(long)},
		{ID: "2", Type: session.EntryMessage, Message: agent.NewUserMessage("recent")},
	}

	tests := []struct {
		name     string
		entries  []session.TreeEntry
		window   int
		wantMsgs int
	}{
		{name: "selects recent within budget", entries: entries, window: 500, wantMsgs: 1},
		{name: "empty entries", entries: nil, window: 1000, wantMsgs: 0},
		{name: "zero budget uses keep recent fallback", entries: entries[:1], window: 0, wantMsgs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			messages, _, tokens := ctxmgr.PrepareBranchSummary(tt.entries, tt.window, ctxmgr.Settings{KeepRecentTokens: 50})
			assert.Len(t, messages, tt.wantMsgs)
			if tt.wantMsgs == 0 {
				assert.Zero(t, tokens)
			}
		})
	}
}

func TestRunBranchSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []agent.AgentMessage
		wantOK   bool
		wantText string
	}{
		{name: "empty messages returns empty summary", messages: nil, wantOK: true, wantText: ""},
		{name: "generates summary text", messages: []agent.AgentMessage{agent.NewUserMessage("discuss plan")}, wantOK: true, wantText: "Branch recap"},
		{name: "multiple messages summarized", messages: []agent.AgentMessage{
			agent.NewUserMessage("one"),
			agent.NewUserMessage("two"),
		}, wantOK: true, wantText: "Branch recap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nBranch recap"})
			result := ctxmgr.RunBranchSummary(
				context.Background(),
				model,
				"fake",
				tt.messages,
				ctxmgr.NewFileOperations(),
				ctxmgr.Settings{},
			)
			require.Equal(t, tt.wantOK, result.IsOk())
			if !tt.wantOK {
				return
			}
			summary := result.Value().Summary
			if tt.wantText == "" {
				assert.Empty(t, summary)
				return
			}
			assert.Contains(t, summary, tt.wantText)
		})
	}
}
