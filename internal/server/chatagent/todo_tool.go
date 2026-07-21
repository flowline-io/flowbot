package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	todoWriteToolName = "todo_write"
	listTodosToolName = "list_todos"
)

// Valid todo status values.
const (
	TodoStatusPending    = "pending"
	TodoStatusInProgress = "in_progress"
	TodoStatusCompleted  = "completed"
	TodoStatusCancelled  = "cancelled"
)

// TodoToolDeps carries per-run metadata for session checklist tools.
type TodoToolDeps struct {
	SessionID string
}

// TodoTools registers todo_write and list_todos.
type TodoTools struct {
	deps TodoToolDeps
}

// NewTodoTools binds checklist tools to one chat run.
func NewTodoTools(deps TodoToolDeps) TodoTools {
	return TodoTools{deps: deps}
}

// Register adds checklist tools to the registry.
func (t TodoTools) Register(registry *tool.Registry) error {
	tools := []tool.Tool{
		TodoWriteTool{deps: t.deps},
		ListTodosTool{deps: t.deps},
	}
	for _, td := range tools {
		if err := registry.Register(td); err != nil {
			return err
		}
	}
	return nil
}

func todoToolNames() []string {
	return []string{todoWriteToolName, listTodosToolName}
}

// TodoWriteTool merges or replaces the session checklist.
type TodoWriteTool struct {
	deps TodoToolDeps
}

func (TodoWriteTool) Name() string { return todoWriteToolName }

func (TodoWriteTool) Description() string {
	return "Create or update the session todo checklist. Use merge=true to upsert items by id; merge=false replaces the entire list."
}

func (TodoWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"merge": map[string]any{
				"type":        "boolean",
				"description": "When true, upsert items by id; when false, replace the entire checklist",
			},
			"todos": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Stable item identifier within the session",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "Todo description",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "One of pending, in_progress, completed, cancelled",
							"enum":        []string{TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled},
						},
					},
					"required": []string{"id", "content", "status"},
				},
				"minItems": 1,
			},
		},
		"required": []string{"todos"},
	}
}

func (t TodoWriteTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if strings.TrimSpace(t.deps.SessionID) == "" {
		return todoToolError(id, todoWriteToolName, "session unavailable"), nil
	}
	if store.Database == nil {
		return todoToolError(id, todoWriteToolName, "store unavailable"), nil
	}
	parsed, errText := parseTodoWriteArgs(args)
	if errText != "" {
		return todoToolError(id, todoWriteToolName, errText), nil
	}
	rows := parsed.toRows(t.deps.SessionID)
	if parsed.merge {
		if err := store.Database.MergeAgentTodosForSession(ctx, t.deps.SessionID, rows); err != nil {
			flog.Warn("[chat-agent] todo_write merge failed session=%s: %v", t.deps.SessionID, err)
			return todoToolError(id, todoWriteToolName, fmt.Sprintf("merge todos: %v", err)), nil
		}
	} else {
		if err := store.Database.ReplaceAgentTodosForSession(ctx, t.deps.SessionID, rows); err != nil {
			flog.Warn("[chat-agent] todo_write replace failed session=%s: %v", t.deps.SessionID, err)
			return todoToolError(id, todoWriteToolName, fmt.Sprintf("replace todos: %v", err)), nil
		}
	}
	return todoSnapshotResult(ctx, id, todoWriteToolName, t.deps.SessionID)
}

// ListTodosTool returns the current session checklist.
type ListTodosTool struct {
	deps TodoToolDeps
}

func (ListTodosTool) Name() string { return listTodosToolName }

func (ListTodosTool) Description() string {
	return "List the current session todo checklist with item id, content, and status."
}

func (ListTodosTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t ListTodosTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if strings.TrimSpace(t.deps.SessionID) == "" {
		return todoToolError(id, listTodosToolName, "session unavailable"), nil
	}
	return todoSnapshotResult(ctx, id, listTodosToolName, t.deps.SessionID)
}

type parsedTodoWrite struct {
	merge bool
	items []parsedTodoItem
}

type parsedTodoItem struct {
	itemID    string
	content   string
	status    string
	sortOrder int
}

func parseTodoWriteArgs(args map[string]any) (parsedTodoWrite, string) {
	merge := true
	if raw, ok := args["merge"].(bool); ok {
		merge = raw
	}
	rawTodos, ok := args["todos"].([]any)
	if !ok || len(rawTodos) == 0 {
		return parsedTodoWrite{}, "todos must be a non-empty array"
	}
	items := make([]parsedTodoItem, 0, len(rawTodos))
	for i, raw := range rawTodos {
		obj, ok := raw.(map[string]any)
		if !ok {
			return parsedTodoWrite{}, fmt.Sprintf("todos[%d] must be an object", i)
		}
		itemID := strings.TrimSpace(stringArg(obj, "id"))
		content := strings.TrimSpace(stringArg(obj, "content"))
		status := strings.TrimSpace(stringArg(obj, "status"))
		if itemID == "" {
			return parsedTodoWrite{}, fmt.Sprintf("todos[%d].id is required", i)
		}
		if content == "" {
			return parsedTodoWrite{}, fmt.Sprintf("todos[%d].content is required", i)
		}
		if !validTodoStatus(status) {
			return parsedTodoWrite{}, fmt.Sprintf("todos[%d].status must be pending, in_progress, completed, or cancelled", i)
		}
		items = append(items, parsedTodoItem{
			itemID:    itemID,
			content:   content,
			status:    status,
			sortOrder: i,
		})
	}
	return parsedTodoWrite{merge: merge, items: items}, ""
}

func (p parsedTodoWrite) toRows(sessionID string) []*gen.AgentTodo {
	rows := make([]*gen.AgentTodo, 0, len(p.items))
	for _, item := range p.items {
		rows = append(rows, &gen.AgentTodo{
			Flag:      types.Id(),
			SessionID: sessionID,
			ItemID:    item.itemID,
			Content:   item.content,
			Status:    item.status,
			SortOrder: item.sortOrder,
		})
	}
	return rows
}

func validTodoStatus(status string) bool {
	switch status {
	case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
		return true
	default:
		return false
	}
}

func todoSnapshotResult(ctx context.Context, id, name, sessionID string) (msg.ToolResultMessage, error) {
	items, err := ListTodoItems(ctx, sessionID)
	if err != nil {
		return todoToolError(id, name, err.Error()), nil
	}
	payload, err := sonic.MarshalString(TodoListSnapshot{Todos: items})
	if err != nil {
		return todoToolError(id, name, "encode todos"), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: payload}},
	}, nil
}

func todoToolError(id, name, message string) msg.ToolResultMessage {
	return tool.ErrorResult(id, name, "todo_error", message, "")
}
