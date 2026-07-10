package partials

import (
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

// AgentScheduledTaskDetailURL builds the detail page URL for a scheduled task.
func AgentScheduledTaskDetailURL(taskID string) templ.SafeURL {
	return templ.URL("/service/web/agent-scheduled-tasks/" + taskID)
}

// AgentScheduledTaskPageTitle returns the browser title for a scheduled task detail page.
func AgentScheduledTaskPageTitle(task model.AgentScheduledTask) string {
	if strings.TrimSpace(task.Name) != "" {
		return task.Name + " — Flowbot"
	}
	return "Scheduled Task " + task.TaskID + " — Flowbot"
}

// AgentScheduledTaskStateBadgeClass returns DaisyUI badge classes for task state.
func AgentScheduledTaskStateBadgeClass(state string) string {
	switch state {
	case "active":
		return "badge badge-success badge-sm"
	case "paused":
		return "badge badge-warning badge-sm"
	case "completed":
		return "badge badge-info badge-sm"
	case "failed", "cancelled", "missed":
		return "badge badge-error badge-sm"
	default:
		return "badge badge-ghost badge-sm"
	}
}

// AgentScheduledTaskRunStateBadgeClass returns DaisyUI badge classes for run state.
func AgentScheduledTaskRunStateBadgeClass(state string) string {
	switch state {
	case "completed":
		return "badge badge-success badge-sm"
	case "running":
		return "badge badge-warning badge-sm"
	case "failed":
		return "badge badge-error badge-sm"
	default:
		return "badge badge-ghost badge-sm"
	}
}

// AgentScheduledTaskKindLabel returns a user-friendly label for schedule kind.
func AgentScheduledTaskKindLabel(kind string) string {
	switch kind {
	case "cron":
		return "Cron"
	case "once":
		return "Once"
	default:
		return kind
	}
}

// AgentScheduledTaskScheduleSummary renders a compact schedule summary.
func AgentScheduledTaskScheduleSummary(task model.AgentScheduledTask) string {
	if strings.EqualFold(task.ScheduleKind, "cron") {
		if strings.TrimSpace(task.Cron) == "" {
			return "-"
		}
		return task.Cron
	}
	if task.RunAt == nil || task.RunAt.IsZero() {
		return "-"
	}
	return task.RunAt.Format("2006-01-02 15:04:05 UTC")
}

// AgentScheduledTaskTimeOrDash formats an optional timestamp.
func AgentScheduledTaskTimeOrDash(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04:05")
}

// AgentScheduledTaskTextPreview truncates long content for table cells.
func AgentScheduledTaskTextPreview(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
	if limit <= 0 {
		limit = 120
	}
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
