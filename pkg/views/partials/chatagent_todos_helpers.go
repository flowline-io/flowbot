package partials

import (
	"context"
	"strconv"
	"strings"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func chatAgentTodosPanelClass(count int) string {
	_ = count
	return "chatagent-todos-panel flowbot-surface shrink-0 mx-1 mb-3"
}

func chatAgentTodosCountLabel(ctx context.Context, todos []model.AgentTodo) string {
	if len(todos) == 0 {
		return i18n.T(ctx, "client.chatagent.todos.count_zero")
	}
	done, total := chatAgentTodosProgress(todos)
	if done == total {
		return i18n.TData(ctx, "client.chatagent.todos.count_done", map[string]any{"Done": done, "Total": total})
	}
	active := 0
	for _, item := range todos {
		switch item.Status {
		case "completed", "cancelled":
			continue
		default:
			active++
		}
	}
	if active <= 0 {
		return i18n.TData(ctx, "client.chatagent.todos.count_done", map[string]any{"Done": done, "Total": total})
	}
	return i18n.TData(ctx, "client.chatagent.todos.count_active", map[string]any{"Active": active, "Done": done, "Total": total})
}

func chatAgentTodosProgressWidth(todos []model.AgentTodo) string {
	done, total := chatAgentTodosProgress(todos)
	if total == 0 {
		return "0%"
	}
	pct := (done * 100) / total
	return strconv.Itoa(pct) + "%"
}

func chatAgentTodosProgressPercentLabel(todos []model.AgentTodo) string {
	done, total := chatAgentTodosProgress(todos)
	if total == 0 {
		return "0%"
	}
	pct := (done * 100) / total
	return strconv.Itoa(pct) + "%"
}

func chatAgentTodosProgress(todos []model.AgentTodo) (done, total int) {
	total = len(todos)
	for _, item := range todos {
		if item.Status == "completed" {
			done++
		}
	}
	return done, total
}

func chatAgentTodoSummaryCountLabel(ctx context.Context, summary model.AgentTodoSummary) string {
	if summary.Total == 0 {
		return i18n.T(ctx, "client.chatagent.todos.count_zero")
	}
	if summary.Done == summary.Total {
		return i18n.TData(ctx, "client.chatagent.todos.count_done", map[string]any{"Done": summary.Done, "Total": summary.Total})
	}
	if summary.Active <= 0 {
		return i18n.TData(ctx, "client.chatagent.todos.count_done", map[string]any{"Done": summary.Done, "Total": summary.Total})
	}
	return i18n.TData(ctx, "client.chatagent.todos.count_active", map[string]any{"Active": summary.Active, "Done": summary.Done, "Total": summary.Total})
}

func chatAgentTodoSummaryProgressWidth(summary model.AgentTodoSummary) string {
	if summary.Total == 0 {
		return "0%"
	}
	pct := (summary.Done * 100) / summary.Total
	return strconv.Itoa(pct) + "%"
}

func chatAgentTodoSummaryProgressPercentLabel(summary model.AgentTodoSummary) string {
	if summary.Total == 0 {
		return "0%"
	}
	pct := (summary.Done * 100) / summary.Total
	return strconv.Itoa(pct) + "%"
}

func chatAgentSessionTodoLineLabel(ctx context.Context, summary model.AgentTodoSummary) string {
	if summary.Total == 0 {
		return ""
	}
	if summary.Done == summary.Total {
		return i18n.TData(ctx, "client.chatagent.todos.count_done", map[string]any{"Done": summary.Done, "Total": summary.Total})
	}
	parts := []string{
		chatAgentTodoSummaryProgressPercentLabel(summary),
		chatAgentTodoSummaryCountLabel(ctx, summary),
	}
	if summary.InProgress != "" {
		parts = append(parts, summary.InProgress)
	}
	return strings.Join(parts, " · ")
}

func chatAgentTodoItemClass(status string) string {
	return "chatagent-todos-item chatagent-todos-item--" + chatAgentTodoStatusSlug(status)
}

func chatAgentTodoMarkerClass(status string) string {
	return "chatagent-todos-marker chatagent-todos-marker--" + chatAgentTodoStatusSlug(status)
}

func chatAgentTodoStatusSlug(status string) string {
	switch status {
	case "in_progress":
		return "in-progress"
	case "completed":
		return "completed"
	case "cancelled":
		return "cancelled"
	default:
		return "pending"
	}
}
