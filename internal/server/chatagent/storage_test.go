package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkBranchFromLeaf(t *testing.T) {
	const sessionID = "sess-1"

	entries := map[string]*gen.ChatSessionEntry{
		"root": {Flag: "root", SessionID: sessionID, ParentID: "", EntryType: "message", Payload: map[string]any{"id": "root", "type": "message"}},
		"m2":   {Flag: "m2", SessionID: sessionID, ParentID: "root", EntryType: "message", Payload: map[string]any{"id": "m2", "type": "message", "parent_id": "root"}},
		"leaf": {Flag: "leaf", SessionID: sessionID, ParentID: "m2", EntryType: "message", Payload: map[string]any{"id": "leaf", "type": "message", "parent_id": "m2"}},
		"dead": {Flag: "dead", SessionID: sessionID, ParentID: "root", EntryType: "message", Payload: map[string]any{"id": "dead", "type": "message", "parent_id": "root"}},
	}

	getter := func(_ context.Context, gotSessionID, flag string) (*gen.ChatSessionEntry, error) {
		require.Equal(t, sessionID, gotSessionID)
		row, ok := entries[flag]
		if !ok {
			return nil, assert.AnError
		}
		return row, nil
	}

	tests := []struct {
		name      string
		leafID    string
		wantIDs   []string
		wantCalls int
	}{
		{
			name:      "walks active branch depth only",
			leafID:    "leaf",
			wantIDs:   []string{"root", "m2", "leaf"},
			wantCalls: 3,
		},
		{
			name:      "single root leaf",
			leafID:    "root",
			wantIDs:   []string{"root"},
			wantCalls: 1,
		},
		{
			name:      "empty leaf id",
			leafID:    "",
			wantIDs:   nil,
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			trackingGetter := func(ctx context.Context, gotSessionID, flag string) (*gen.ChatSessionEntry, error) {
				calls++
				return getter(ctx, gotSessionID, flag)
			}

			got, err := walkBranchFromLeaf(context.Background(), sessionID, tt.leafID, trackingGetter)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCalls, calls)

			if tt.wantIDs == nil {
				assert.Nil(t, got)
				return
			}
			require.Len(t, got, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, got[i].ID)
			}
		})
	}
}

func TestWalkBranchFromLeafBrokenChain(t *testing.T) {
	t.Parallel()
	getter := func(_ context.Context, _, flag string) (*gen.ChatSessionEntry, error) {
		if flag == "leaf" {
			return &gen.ChatSessionEntry{Flag: "leaf", ParentID: "missing"}, nil
		}
		return nil, assert.AnError
	}
	_, err := walkBranchFromLeaf(context.Background(), "sess", "leaf", getter)
	require.Error(t, err)
}

func TestWalkBranchFromLeafDoesNotLoadDeadEntries(t *testing.T) {
	t.Parallel()
	const sessionID = "sess-1"
	entries := map[string]*gen.ChatSessionEntry{
		"root": {Flag: "root", SessionID: sessionID, ParentID: "", EntryType: "message", Payload: map[string]any{"id": "root", "type": "message"}},
		"leaf": {Flag: "leaf", SessionID: sessionID, ParentID: "root", EntryType: "message", Payload: map[string]any{"id": "leaf", "type": "message", "parent_id": "root"}},
	}
	loaded := make(map[string]struct{})
	getter := func(_ context.Context, _, flag string) (*gen.ChatSessionEntry, error) {
		loaded[flag] = struct{}{}
		row, ok := entries[flag]
		if !ok {
			return nil, assert.AnError
		}
		return row, nil
	}

	branch, err := walkBranchFromLeaf(context.Background(), sessionID, "leaf", getter)
	require.NoError(t, err)
	require.Len(t, branch, 2)
	assert.Contains(t, loaded, "root")
	assert.Contains(t, loaded, "leaf")
	assert.NotContains(t, loaded, "dead")
}
