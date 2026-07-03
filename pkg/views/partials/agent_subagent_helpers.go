package partials

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func agentSubagentRowID(item model.AgentSubagent) string {
	return "agent-subagent-" + url.PathEscape(item.Flag)
}

func agentSubagentFormID(item model.AgentSubagent, isNew bool) string {
	if isNew {
		return "agent-subagent-form-new"
	}
	return "agent-subagent-form-" + agentSubagentRowID(item)
}

func agentSubagentURL(item model.AgentSubagent) string {
	return fmt.Sprintf("/service/web/agent-subagents/%s", url.PathEscape(item.Flag))
}

func agentSubagentEditURL(item model.AgentSubagent) string {
	return agentSubagentURL(item) + "/edit"
}

func agentSubagentListURL() string {
	return "/service/web/agent-subagents/list"
}

func agentSubagentCancelURL() string {
	return agentSubagentListURL()
}

func agentSubagentDescriptionPreview(description string) string {
	if len(description) <= 60 {
		return description
	}
	return description[:57] + "..."
}

func agentSubagentModelLabel(modelName string) string {
	if strings.TrimSpace(modelName) == "" {
		return "(default)"
	}
	return modelName
}

func agentSubagentOptionSelected(selected []string, value string) bool {
	return slices.Contains(selected, value)
}

func agentSubagentTaskRowID(item model.AgentSubagentTask) string {
	return fmt.Sprintf("agent-subagent-task-%d", item.ID)
}

func agentSubagentTaskDetailID(item model.AgentSubagentTask) string {
	return fmt.Sprintf("agent-subagent-task-detail-%d", item.ID)
}

func agentSubagentTaskDetailURL(item model.AgentSubagentTask) string {
	return fmt.Sprintf("/service/web/agent-subagents/tasks/%d", item.ID)
}

func agentSubagentTasksListURL() string {
	return "/service/web/agent-subagents/tasks"
}

func agentSubagentTaskDescriptionPreview(description string) string {
	if strings.TrimSpace(description) == "" {
		return "(no description)"
	}
	if len(description) <= 60 {
		return description
	}
	return description[:57] + "..."
}

func agentSubagentTaskStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "running":
		return "Running"
	case "completed":
		return "Completed"
	case "failed":
		return "Failed"
	default:
		return status
	}
}

func agentSubagentTaskStatusBadgeClass(status string) string {
	switch strings.TrimSpace(status) {
	case "running":
		return "badge badge-warning badge-sm"
	case "completed":
		return "badge badge-success badge-sm"
	case "failed":
		return "badge badge-error badge-sm"
	default:
		return "badge badge-ghost badge-sm"
	}
}

func agentSubagentTaskDuration(item model.AgentSubagentTask) string {
	if item.FinishedAt == nil {
		return "—"
	}
	d := item.FinishedAt.Sub(item.StartedAt)
	if d < 0 {
		return "—"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(time.Second).String()
}
