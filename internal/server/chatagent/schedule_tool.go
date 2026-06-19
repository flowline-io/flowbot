package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ScheduleToolDeps carries per-run metadata for scheduled task tools.
type ScheduleToolDeps struct {
	UID       types.Uid
	SessionID string
}

// ScheduleTools registers create/list/update/cancel scheduled task tools.
type ScheduleTools struct {
	deps ScheduleToolDeps
}

// NewScheduleTools binds scheduled task tools to one chat run.
func NewScheduleTools(deps ScheduleToolDeps) ScheduleTools {
	return ScheduleTools{deps: deps}
}

// Register adds all schedule tools to the registry.
func (s ScheduleTools) Register(registry *tool.Registry) error {
	tools := []tool.Tool{
		ScheduleTaskTool{deps: s.deps},
		UpdateScheduledTaskTool{deps: s.deps},
		ListScheduledTasksTool{deps: s.deps},
		CancelScheduledTaskTool{deps: s.deps},
	}
	for _, t := range tools {
		if err := registry.Register(t); err != nil {
			return err
		}
	}
	return nil
}

// ScheduleTaskTool creates cron or one-shot scheduled tasks.
type ScheduleTaskTool struct {
	deps ScheduleToolDeps
}

func (ScheduleTaskTool) Name() string { return scheduleToolName }

func (ScheduleTaskTool) Description() string {
	return "Creates a scheduled chat agent task. Provide cron for recurring jobs or run_at (ISO8601 UTC) for one-shot jobs, plus name and prompt."
}

func (ScheduleTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Short (3-5 word) task label",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Self-contained instruction for the agent when the task fires",
			},
			"cron": map[string]any{
				"type":        "string",
				"description": "Cron expression for recurring tasks (5-field, e.g. 0 9 * * *)",
			},
			"run_at": map[string]any{
				"type":        "string",
				"description": "One-shot trigger time in ISO8601 UTC (e.g. 2026-06-20T09:00:00Z)",
			},
		},
		"required": []string{"name", "prompt"},
	}
}

func (t ScheduleTaskTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	parsed := parseCreateScheduleArgs(args)
	if parsed.errText != "" {
		return scheduleToolError(id, scheduleToolName, parsed.errText), nil
	}
	if store.Database == nil {
		return scheduleToolError(id, scheduleToolName, "store unavailable"), nil
	}

	task, err := persistScheduledTask(ctx, t.deps, parsed)
	if err != nil {
		flog.Warn("[chat-agent] schedule_task create failed: %v", err)
		return scheduleToolError(id, scheduleToolName, fmt.Sprintf("create task: %v", err)), nil
	}

	text := fmt.Sprintf("Scheduled task %q created (id=%s, kind=%s, schedule=%s)",
		task.Name, task.Flag, task.ScheduleKind, scheduleDescription(task))
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       scheduleToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

// UpdateScheduledTaskTool modifies cron, run_at, prompt, or name.
type UpdateScheduledTaskTool struct {
	deps ScheduleToolDeps
}

func (UpdateScheduledTaskTool) Name() string { return updateScheduleToolName }

func (UpdateScheduledTaskTool) Description() string {
	return "Updates an existing scheduled task. Requires task_id and at least one of cron, run_at, prompt, name, or state (active|paused)."
}

func (UpdateScheduledTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task id from list_scheduled_tasks",
			},
			"name":   map[string]any{"type": "string"},
			"prompt": map[string]any{"type": "string"},
			"cron":   map[string]any{"type": "string"},
			"run_at": map[string]any{
				"type":        "string",
				"description": "New one-shot time in ISO8601 UTC",
			},
			"state": map[string]any{
				"type":        "string",
				"description": "Set to active or paused",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t UpdateScheduledTaskTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	parsed := parseUpdateScheduleArgs(args)
	if parsed.errText != "" {
		return scheduleToolError(id, updateScheduleToolName, parsed.errText), nil
	}
	if store.Database == nil {
		return scheduleToolError(id, updateScheduleToolName, "store unavailable"), nil
	}

	task, err := store.Database.GetChatScheduledTaskForUID(ctx, parsed.taskID, t.deps.UID.String())
	if err != nil {
		return scheduleToolError(id, updateScheduleToolName, fmt.Sprintf("task not found: %v", err)), nil
	}
	if task.State != string(schema.ChatScheduledTaskStateActive) && task.State != string(schema.ChatScheduledTaskStatePaused) {
		return scheduleToolError(id, updateScheduleToolName, "task is not editable"), nil
	}

	params, errText := buildTaskUpdateParams(task, parsed)
	if errText != "" {
		return scheduleToolError(id, updateScheduleToolName, errText), nil
	}
	updated, err := applyScheduledTaskUpdate(ctx, parsed.taskID, params)
	if err != nil {
		return scheduleToolError(id, updateScheduleToolName, fmt.Sprintf("update task: %v", err)), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       updateScheduleToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: fmt.Sprintf("Updated scheduled task %q (id=%s, state=%s)", updated.Name, parsed.taskID, updated.State)}},
	}, nil
}

// ListScheduledTasksTool lists active and paused tasks for the user.
type ListScheduledTasksTool struct {
	deps ScheduleToolDeps
}

func (ListScheduledTasksTool) Name() string { return listScheduleToolName }

func (ListScheduledTasksTool) Description() string {
	return "Lists the user's active and paused scheduled tasks with ids, schedules, and prompt summaries."
}

func (ListScheduledTasksTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t ListScheduledTasksTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return scheduleToolError(id, listScheduleToolName, "store unavailable"), nil
	}
	tasks, err := store.Database.ListChatScheduledTasks(ctx, store.ListChatScheduledTasksOptions{
		UID: t.deps.UID.String(),
		States: []string{
			string(schema.ChatScheduledTaskStateActive),
			string(schema.ChatScheduledTaskStatePaused),
		},
	})
	if err != nil {
		return scheduleToolError(id, listScheduleToolName, fmt.Sprintf("list tasks: %v", err)), nil
	}
	if len(tasks) == 0 {
		return msg.ToolResultMessage{
			ToolCallID: id,
			Name:       listScheduleToolName,
			Parts:      []msg.ContentPart{msg.TextPart{Text: "No scheduled tasks."}},
		}, nil
	}
	lines := make([]string, 0, len(tasks))
	for _, task := range tasks {
		lines = append(lines, formatScheduledTaskListLine(task))
	}
	text := strings.Join(lines, "\n")
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       listScheduleToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

// CancelScheduledTaskTool cancels a scheduled task.
type CancelScheduledTaskTool struct {
	deps ScheduleToolDeps
}

func (CancelScheduledTaskTool) Name() string { return cancelScheduleToolName }

func (CancelScheduledTaskTool) Description() string {
	return "Cancels a scheduled task by task_id."
}

func (CancelScheduledTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{"type": "string"},
		},
		"required": []string{"task_id"},
	}
}

func (t CancelScheduledTaskTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	taskID := stringArg(args, "task_id")
	if taskID == "" {
		return scheduleToolError(id, cancelScheduleToolName, errTaskIDRequired.Error()), nil
	}
	if store.Database == nil {
		return scheduleToolError(id, cancelScheduleToolName, "store unavailable"), nil
	}
	if _, err := store.Database.GetChatScheduledTaskForUID(ctx, taskID, t.deps.UID.String()); err != nil {
		return scheduleToolError(id, cancelScheduleToolName, fmt.Sprintf("task not found: %v", err)), nil
	}
	cancelled := string(schema.ChatScheduledTaskStateCancelled)
	if err := store.Database.UpdateChatScheduledTask(ctx, taskID, store.UpdateChatScheduledTaskParams{
		State: &cancelled,
	}); err != nil {
		return scheduleToolError(id, cancelScheduleToolName, fmt.Sprintf("cancel task: %v", err)), nil
	}
	if sched := GlobalScheduler(); sched != nil {
		sched.UnregisterTask(taskID)
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       cancelScheduleToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: fmt.Sprintf("Cancelled scheduled task %s", taskID)}},
	}, nil
}

func scheduleToolError(id, name, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    true,
	}
}

func scheduleToolNames() []string {
	return []string{
		scheduleToolName,
		updateScheduleToolName,
		listScheduleToolName,
		cancelScheduleToolName,
	}
}
