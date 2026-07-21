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

func withTestTodoStore(t *testing.T, fn func()) {
	t.Helper()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	defer func() { store.Database = origDB }()
	fn()
}

func TestTodoWriteToolValidation(t *testing.T) {
	t.Parallel()
	tool := TodoWriteTool{deps: TodoToolDeps{SessionID: "sess-1"}}

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{name: "missing todos", args: map[string]any{"merge": true}, wantErr: true},
		{name: "empty todos array", args: map[string]any{"todos": []any{}}, wantErr: true},
		{name: "missing item id", args: map[string]any{"todos": []any{map[string]any{"content": "x", "status": TodoStatusPending}}}, wantErr: true},
		{name: "invalid status", args: map[string]any{"todos": []any{map[string]any{"id": "1", "content": "x", "status": "done"}}}, wantErr: true},
		{name: "valid merge payload", args: map[string]any{"merge": true, "todos": []any{map[string]any{"id": "1", "content": "Plan work", "status": TodoStatusPending}}}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			withTestTodoStore(t, func() {
				result, err := tool.Execute(context.Background(), "call-1", tt.args, nil)
				require.NoError(t, err)
				if tt.wantErr {
					assert.True(t, result.IsError)
					return
				}
				assert.False(t, result.IsError)
			})
		})
	}
}

func TestTodoWriteToolMergeAndReplace(t *testing.T) {
	withTestTodoStore(t, func() {
		writeTool := TodoWriteTool{deps: TodoToolDeps{SessionID: "sess-todo"}}
		listTool := ListTodosTool{deps: TodoToolDeps{SessionID: "sess-todo"}}

		_, err := writeTool.Execute(context.Background(), "call-1", map[string]any{
			"merge": false,
			"todos": []any{
				map[string]any{"id": "a", "content": "First", "status": TodoStatusPending},
				map[string]any{"id": "b", "content": "Second", "status": TodoStatusInProgress},
			},
		}, nil)
		require.NoError(t, err)

		_, err = writeTool.Execute(context.Background(), "call-2", map[string]any{
			"merge": true,
			"todos": []any{
				map[string]any{"id": "a", "content": "First", "status": TodoStatusCompleted},
				map[string]any{"id": "c", "content": "Third", "status": TodoStatusPending},
			},
		}, nil)
		require.NoError(t, err)

		result, err := listTool.Execute(context.Background(), "call-3", nil, nil)
		require.NoError(t, err)
		require.False(t, result.IsError)
		text := todoToolResultText(result)
		assert.Contains(t, text, `"item_id":"a"`)
		assert.Contains(t, text, `"status":"completed"`)
		assert.Contains(t, text, `"item_id":"b"`)
		assert.Contains(t, text, `"item_id":"c"`)
	})
}

func TestTodoWriteToolReplaceClearsMissingItems(t *testing.T) {
	withTestTodoStore(t, func() {
		writeTool := TodoWriteTool{deps: TodoToolDeps{SessionID: "sess-replace"}}

		_, err := writeTool.Execute(context.Background(), "call-1", map[string]any{
			"merge": false,
			"todos": []any{
				map[string]any{"id": "keep", "content": "Keep", "status": TodoStatusPending},
			},
		}, nil)
		require.NoError(t, err)

		items, err := ListTodoItems(context.Background(), "sess-replace")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "keep", items[0].ItemID)
	})
}

func todoToolResultText(result msg.ToolResultMessage) string {
	for _, part := range result.Parts {
		if text, ok := part.(msg.TextPart); ok {
			return text.Text
		}
	}
	return ""
}

func TestValidTodoStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "pending", status: TodoStatusPending, want: true},
		{name: "in progress", status: TodoStatusInProgress, want: true},
		{name: "completed", status: TodoStatusCompleted, want: true},
		{name: "cancelled", status: TodoStatusCancelled, want: true},
		{name: "invalid", status: "done", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, validTodoStatus(tt.status))
		})
	}
}
