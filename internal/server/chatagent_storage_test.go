package server

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBStorageGetBranchBrokenChain(t *testing.T) {
	setupSQLiteTestDB(t)
	seedTestChatSession(t, &gen.ChatSession{Flag: "s1", LeafID: "leaf"})
	seedTestChatSessionEntry(t, &gen.ChatSessionEntry{
		Flag: "leaf", SessionID: "s1", ParentID: "missing", EntryType: "message",
		Payload: map[string]any{"id": "leaf", "parentId": "missing", "type": "message"},
	})

	storage := chatagent.NewDBStorage("s1", types.Uid(""), "")
	_, err := storage.GetBranch(context.Background(), "leaf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load entry")
}
