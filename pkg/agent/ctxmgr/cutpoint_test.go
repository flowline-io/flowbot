package ctxmgr_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindCutPoint(t *testing.T) {
	longText := agent.NewUserMessage(string(make([]byte, 400)))
	entries := []session.TreeEntry{
		{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("old")},
		{ID: "2", Type: session.EntryMessage, Message: longText},
		{ID: "3", Type: session.EntryMessage, Message: agent.NewUserMessage("recent")},
	}

	tests := []struct {
		name       string
		entries    []session.TreeEntry
		start      int
		end        int
		keepRecent int
		wantFirst  string
	}{
		{name: "keeps recent within budget", entries: entries, start: 0, end: 3, keepRecent: 10, wantFirst: "2"},
		{name: "empty range", entries: entries, start: 0, end: 0, keepRecent: 100, wantFirst: "1"},
		{name: "full keep", entries: entries[:1], start: 0, end: 1, keepRecent: 100000, wantFirst: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.FindCutPoint(tt.entries, tt.start, tt.end, tt.keepRecent)
			require.NotEmpty(t, tt.entries)
			assert.Equal(t, tt.wantFirst, tt.entries[got.FirstKeptEntryIndex].ID)
		})
	}
}

func TestPrepareCompaction(t *testing.T) {
	entries := []session.TreeEntry{
		{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("one")},
		{ID: "2", Type: session.EntryMessage, Message: agent.NewUserMessage("two")},
		{ID: "3", Type: session.EntryMessage, Message: agent.NewUserMessage("three")},
	}

	tests := []struct {
		name      string
		entries   []session.TreeEntry
		force     bool
		keepRecent int
		wantNil   bool
		wantFirst string
	}{
		{name: "normal path", entries: entries, keepRecent: 2, wantFirst: "3"},
		{name: "already compacted leaf", entries: append(entries, session.TreeEntry{ID: "4", Type: session.EntryCompaction, Summary: "done", FirstKeptEntryID: "1"}), wantNil: true},
		{
			name: "force re-compact leaf",
			entries: append(entries, session.TreeEntry{
				ID: "4", Type: session.EntryCompaction, Summary: "done", FirstKeptEntryID: "2",
			}),
			force:     true,
			keepRecent: 100000,
			wantFirst: "3",
		},
		{name: "single entry fits budget", entries: entries[:1], keepRecent: 100000, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			keepRecent := tt.keepRecent
			if keepRecent == 0 {
				keepRecent = 100000
			}
			got, err := ctxmgr.PrepareCompaction(tt.entries, ctxmgr.Settings{Enabled: true, KeepRecentTokens: keepRecent}, ctxmgr.PrepareOptions{Force: tt.force})
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantFirst, got.FirstKeptEntryID)
		})
	}
}
