package chatagent

import (
	"context"
	"fmt"
	"slices"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// TodoItem is one checklist row returned by HTTP and tool clients.
type TodoItem struct {
	ItemID    string `json:"item_id"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	SortOrder int    `json:"sort_order"`
}

// TodoListSnapshot is the JSON shape emitted by todo tools and embedded in the Web UI.
type TodoListSnapshot struct {
	Todos []TodoItem `json:"todos"`
}

// ListTodoItems loads the current checklist for one session.
func ListTodoItems(ctx context.Context, sessionID string) ([]TodoItem, error) {
	if store.Database == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	rows, err := store.Database.ListAgentTodosBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return mapTodoItems(rows), nil
}

// ListTodoModels loads checklist rows mapped for server-rendered views.
func ListTodoModels(ctx context.Context, sessionID string) ([]model.AgentTodo, error) {
	items, err := ListTodoItems(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]model.AgentTodo, len(items))
	for i, item := range items {
		out[i] = model.AgentTodo{
			ItemID:    item.ItemID,
			Content:   item.Content,
			Status:    item.Status,
			SortOrder: item.SortOrder,
		}
	}
	return out, nil
}

// SummarizeTodosBySessions returns checklist progress for each session that has todos.
func SummarizeTodosBySessions(ctx context.Context, sessionIDs []string) (map[string]model.AgentTodoSummary, error) {
	if len(sessionIDs) == 0 {
		return map[string]model.AgentTodoSummary{}, nil
	}
	if store.Database == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	rows, err := store.Database.ListAgentTodosBySessions(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}
	bySession := make(map[string][]*gen.AgentTodo, len(sessionIDs))
	for _, row := range rows {
		if row == nil {
			continue
		}
		bySession[row.SessionID] = append(bySession[row.SessionID], row)
	}
	out := make(map[string]model.AgentTodoSummary, len(bySession))
	for sessionID, sessionRows := range bySession {
		summary := summarizeTodoRows(sessionRows)
		if summary.Total > 0 {
			out[sessionID] = summary
		}
	}
	return out, nil
}

func summarizeTodoRows(rows []*gen.AgentTodo) model.AgentTodoSummary {
	summary := model.AgentTodoSummary{Total: len(rows)}
	var inProgress *gen.AgentTodo
	for _, row := range rows {
		if row == nil {
			continue
		}
		switch row.Status {
		case TodoStatusCompleted:
			summary.Done++
		case TodoStatusCancelled:
			continue
		default:
			summary.Active++
			if row.Status == TodoStatusInProgress {
				if inProgress == nil || row.SortOrder < inProgress.SortOrder ||
					(row.SortOrder == inProgress.SortOrder && row.ItemID < inProgress.ItemID) {
					inProgress = row
				}
			}
		}
	}
	if inProgress != nil {
		summary.InProgress = inProgress.Content
	}
	return summary
}

func mapTodoItems(rows []*gen.AgentTodo) []TodoItem {
	out := make([]TodoItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, TodoItem{
			ItemID:    row.ItemID,
			Content:   row.Content,
			Status:    row.Status,
			SortOrder: row.SortOrder,
		})
	}
	slices.SortFunc(out, func(a, b TodoItem) int {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder - b.SortOrder
		}
		if a.ItemID < b.ItemID {
			return -1
		}
		if a.ItemID > b.ItemID {
			return 1
		}
		return 0
	})
	return out
}
