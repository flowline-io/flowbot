package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func memoryToolResultText(result msg.ToolResultMessage) string {
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

func TestMemoryFactToolsCRUD(t *testing.T) {
	orig := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = orig })

	ctx := WithMemoryScope(context.Background(), "default")
	tests := []struct {
		name     string
		run      func(t *testing.T)
		wantText string
		wantErr  bool
	}{
		{
			name: "set fact",
			run: func(t *testing.T) {
				result, err := (MemorySetTool{}).Execute(ctx, "id-1", map[string]any{
					"key": "user.name", "value": "Robin", "pinned": true,
				}, nil)
				require.NoError(t, err)
				assert.False(t, result.IsError)
				assert.Contains(t, memoryToolResultText(result), "user.name")
			},
		},
		{
			name: "get fact",
			run: func(t *testing.T) {
				_, err := (MemorySetTool{}).Execute(ctx, "id-0", map[string]any{
					"key": "user.name", "value": "Robin",
				}, nil)
				require.NoError(t, err)
				result, err := (MemoryGetTool{}).Execute(ctx, "id-1", map[string]any{"key": "user.name"}, nil)
				require.NoError(t, err)
				assert.False(t, result.IsError)
				assert.Contains(t, memoryToolResultText(result), "Robin")
			},
		},
		{
			name: "list facts",
			run: func(t *testing.T) {
				_, err := (MemorySetTool{}).Execute(ctx, "id-0", map[string]any{
					"key": "pref.lang", "value": "zh",
				}, nil)
				require.NoError(t, err)
				result, err := (MemoryListTool{}).Execute(ctx, "id-1", nil, nil)
				require.NoError(t, err)
				assert.False(t, result.IsError)
				assert.Contains(t, memoryToolResultText(result), "pref.lang")
			},
		},
		{
			name: "delete fact",
			run: func(t *testing.T) {
				_, err := (MemorySetTool{}).Execute(ctx, "id-0", map[string]any{
					"key": "tmp.k", "value": "v",
				}, nil)
				require.NoError(t, err)
				result, err := (MemoryDeleteTool{}).Execute(ctx, "id-1", map[string]any{"key": "tmp.k"}, nil)
				require.NoError(t, err)
				assert.False(t, result.IsError)
			},
		},
		{
			name: "set rejects bad key",
			run: func(t *testing.T) {
				result, err := (MemorySetTool{}).Execute(ctx, "id-1", map[string]any{
					"key": "bad key!", "value": "x",
				}, nil)
				require.NoError(t, err)
				assert.True(t, result.IsError)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestSearchSessionSummariesTool(t *testing.T) {
	orig := store.Database
	db := postgres.NewSQLiteTestAdapter(t)
	store.Database = db
	t.Cleanup(func() { store.Database = orig })

	ctx := WithMemoryScope(context.Background(), "default")
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "empty query errors",
			run: func(t *testing.T) {
				result, err := (SearchSessionSummariesTool{}).Execute(ctx, "id", map[string]any{}, nil)
				require.NoError(t, err)
				assert.True(t, result.IsError)
			},
		},
		{
			name: "finds ready summary",
			run: func(t *testing.T) {
				_, err := db.UpsertAgentSessionSummaryPending(ctx, "sess-a", "default", "Widgets")
				require.NoError(t, err)
				_, err = db.ClaimAgentSessionSummaryPending(ctx, "search-tok")
				require.NoError(t, err)
				require.NoError(t, db.MarkAgentSessionSummaryReady(ctx, "sess-a", "search-tok", "Widgets", "talked about widgets"))
				result, err := (SearchSessionSummariesTool{}).Execute(ctx, "id", map[string]any{"query": "widgets"}, nil)
				require.NoError(t, err)
				assert.False(t, result.IsError)
				assert.Contains(t, memoryToolResultText(result), "sess-a")
			},
		},
		{
			name: "no matches returns empty array",
			run: func(t *testing.T) {
				result, err := (SearchSessionSummariesTool{}).Execute(ctx, "id", map[string]any{"query": "zzzz-none"}, nil)
				require.NoError(t, err)
				assert.False(t, result.IsError)
				assert.Equal(t, "[]", memoryToolResultText(result))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
