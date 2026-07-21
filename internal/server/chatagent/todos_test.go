package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeTodosBySessions(t *testing.T) {
	tests := []struct {
		name        string
		sessionIDs  []string
		seed        func(ctx context.Context) error
		wantCounts  map[string]model.AgentTodoSummary
		wantMissing []string
	}{
		{
			name:       "empty session ids",
			sessionIDs: nil,
			wantCounts: map[string]model.AgentTodoSummary{},
		},
		{
			name:       "single session progress",
			sessionIDs: []string{"sess-a"},
			seed: func(ctx context.Context) error {
				return store.Database.ReplaceAgentTodosForSession(ctx, "sess-a", []*gen.AgentTodo{
					{Flag: types.Id(), SessionID: "sess-a", ItemID: "1", Content: "Plan", Status: TodoStatusCompleted, SortOrder: 0},
					{Flag: types.Id(), SessionID: "sess-a", ItemID: "2", Content: "Build", Status: TodoStatusInProgress, SortOrder: 1},
					{Flag: types.Id(), SessionID: "sess-a", ItemID: "3", Content: "Ship", Status: TodoStatusPending, SortOrder: 2},
				})
			},
			wantCounts: map[string]model.AgentTodoSummary{
				"sess-a": {Total: 3, Done: 1, Active: 2, InProgress: "Build"},
			},
		},
		{
			name:       "cancelled items excluded from active count",
			sessionIDs: []string{"sess-b"},
			seed: func(ctx context.Context) error {
				return store.Database.ReplaceAgentTodosForSession(ctx, "sess-b", []*gen.AgentTodo{
					{Flag: types.Id(), SessionID: "sess-b", ItemID: "1", Content: "Done", Status: TodoStatusCompleted, SortOrder: 0},
					{Flag: types.Id(), SessionID: "sess-b", ItemID: "2", Content: "Skip", Status: TodoStatusCancelled, SortOrder: 1},
				})
			},
			wantCounts: map[string]model.AgentTodoSummary{
				"sess-b": {Total: 2, Done: 1, Active: 0},
			},
		},
		{
			name:        "session without todos omitted",
			sessionIDs:  []string{"sess-empty"},
			wantCounts:  map[string]model.AgentTodoSummary{},
			wantMissing: []string{"sess-empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestTodoStore(t, func() {
				ctx := context.Background()
				if tt.seed != nil {
					require.NoError(t, tt.seed(ctx))
				}
				got, err := SummarizeTodosBySessions(ctx, tt.sessionIDs)
				require.NoError(t, err)
				for sessionID, want := range tt.wantCounts {
					assert.Equal(t, want, got[sessionID])
				}
				for _, sessionID := range tt.wantMissing {
					_, ok := got[sessionID]
					assert.False(t, ok)
				}
			})
		})
	}
}
func TestListTodoItemsOrdersBySortOrder(t *testing.T) {
	withTestTodoStore(t, func() {
		ctx := context.Background()
		require.NoError(t, store.Database.ReplaceAgentTodosForSession(ctx, "sess-order", []*gen.AgentTodo{
			{Flag: types.Id(), SessionID: "sess-order", ItemID: "b", Content: "Second", Status: TodoStatusPending, SortOrder: 1},
			{Flag: types.Id(), SessionID: "sess-order", ItemID: "a", Content: "First", Status: TodoStatusPending, SortOrder: 0},
		}))

		items, err := ListTodoItems(ctx, "sess-order")
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "a", items[0].ItemID)
		assert.Equal(t, "b", items[1].ItemID)
	})
}
