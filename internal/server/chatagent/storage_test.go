package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkBranchFromEntries(t *testing.T) {
	t.Parallel()

	entries := []session.TreeEntry{
		{ID: "root", Type: session.EntryMessage, ParentID: ""},
		{ID: "m2", Type: session.EntryMessage, ParentID: "root"},
		{ID: "leaf", Type: session.EntryMessage, ParentID: "m2"},
		{ID: "dead", Type: session.EntryMessage, ParentID: "root"},
	}

	tests := []struct {
		name    string
		leafID  string
		wantIDs []string
	}{
		{
			name:    "walks active branch depth only",
			leafID:  "leaf",
			wantIDs: []string{"root", "m2", "leaf"},
		},
		{
			name:    "single root leaf",
			leafID:  "root",
			wantIDs: []string{"root"},
		},
		{
			name:    "empty leaf id",
			leafID:  "",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := walkBranchFromEntries(entries, tt.leafID)
			require.NoError(t, err)

			if tt.wantIDs == nil {
				assert.Nil(t, got)
				return
			}
			require.Len(t, got, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, got[i].ID)
			}
			for _, entry := range got {
				assert.NotEqual(t, "dead", entry.ID)
			}
		})
	}
}

func TestWalkBranchFromEntriesBrokenChain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []session.TreeEntry
		leafID  string
	}{
		{
			name: "missing parent",
			entries: []session.TreeEntry{
				{ID: "leaf", Type: session.EntryMessage, ParentID: "missing"},
			},
			leafID: "leaf",
		},
		{
			name:    "missing leaf",
			entries: []session.TreeEntry{{ID: "root", Type: session.EntryMessage}},
			leafID:  "leaf",
		},
		{
			name: "broken mid chain",
			entries: []session.TreeEntry{
				{ID: "leaf", Type: session.EntryMessage, ParentID: "gap"},
				{ID: "root", Type: session.EntryMessage, ParentID: ""},
			},
			leafID: "leaf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := walkBranchFromEntries(tt.entries, tt.leafID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "load entry")
		})
	}
}
