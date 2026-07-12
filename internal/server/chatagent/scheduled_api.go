package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ScheduledTaskView is the API representation of one scheduled task.
type ScheduledTaskView struct {
	TaskID          string     `json:"task_id"`
	Name            string     `json:"name"`
	ScheduleKind    string     `json:"schedule_kind"`
	Cron            string     `json:"cron,omitempty"`
	RunAt           *time.Time `json:"run_at,omitempty"`
	Prompt          string     `json:"prompt"`
	State           string     `json:"state"`
	SourceSessionID string     `json:"source_session_id,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	NextRunAt       *time.Time `json:"next_run_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ScheduledTaskRunView is the API representation of one task execution.
type ScheduledTaskRunView struct {
	RunID        string     `json:"run_id"`
	TaskID       string     `json:"task_id"`
	RunSessionID string     `json:"run_session_id"`
	State        string     `json:"state"`
	Reply        string     `json:"reply,omitempty"`
	Error        string     `json:"error,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

// CreateScheduledTaskRequest carries HTTP POST fields for a new scheduled task.
type CreateScheduledTaskRequest struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
	Cron   string `json:"cron,omitempty"`
	RunAt  string `json:"run_at,omitempty"`
}

// UpdateScheduledTaskRequest carries HTTP PATCH fields.
type UpdateScheduledTaskRequest struct {
	Name   *string `json:"name,omitempty"`
	Prompt *string `json:"prompt,omitempty"`
	Cron   *string `json:"cron,omitempty"`
	RunAt  *string `json:"run_at,omitempty"`
	State  *string `json:"state,omitempty"`
}

// ListScheduledTasksForUID returns scheduled tasks owned by uid.
func ListScheduledTasksForUID(ctx context.Context, uid types.Uid, states []string) ([]ScheduledTaskView, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	if len(states) == 0 {
		states = []string{
			string(schema.ChatScheduledTaskStateActive),
			string(schema.ChatScheduledTaskStatePaused),
		}
	}
	rows, err := store.Database.ListChatScheduledTasks(ctx, store.ListChatScheduledTasksOptions{
		UID:    uid.String(),
		States: states,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ScheduledTaskView, 0, len(rows))
	for _, row := range rows {
		out = append(out, taskViewFromRow(row))
	}
	return out, nil
}

// GetScheduledTaskForUID loads one task when owned by uid.
func GetScheduledTaskForUID(ctx context.Context, uid types.Uid, taskID string) (*ScheduledTaskView, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	row, err := store.Database.GetChatScheduledTaskForUID(ctx, taskID, uid.String())
	if err != nil {
		return nil, err
	}
	view := taskViewFromRow(row)
	return &view, nil
}

// CreateScheduledTaskForUID creates one owned scheduled task.
func CreateScheduledTaskForUID(ctx context.Context, uid types.Uid, sourceSessionID string, req CreateScheduledTaskRequest) (*ScheduledTaskView, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	parsed := parseCreateScheduleArgs(map[string]any{
		"name":   req.Name,
		"prompt": req.Prompt,
		"cron":   req.Cron,
		"run_at": req.RunAt,
	})
	if parsed.errText != "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s", parsed.errText)
	}
	task, err := persistScheduledTask(ctx, ScheduleToolDeps{
		UID:       uid,
		SessionID: sourceSessionID,
	}, parsed)
	if err != nil {
		return nil, err
	}
	view := taskViewFromRow(task)
	return &view, nil
}

// SetScheduledTaskStateForUID sets the lifecycle state of one owned task.
func SetScheduledTaskStateForUID(ctx context.Context, uid types.Uid, taskID string, state string) (*ScheduledTaskView, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	state = strings.TrimSpace(state)
	if !ValidScheduledTaskState(state) {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid state")
	}
	if _, err := store.Database.GetChatScheduledTaskForUID(ctx, taskID, uid.String()); err != nil {
		return nil, err
	}
	updated, err := applyScheduledTaskUpdate(ctx, taskID, store.UpdateChatScheduledTaskParams{
		State: &state,
	})
	if err != nil {
		return nil, err
	}
	view := taskViewFromRow(updated)
	return &view, nil
}

// CancelScheduledTaskForUID cancels one owned task.
func CancelScheduledTaskForUID(ctx context.Context, uid types.Uid, taskID string) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	if _, err := store.Database.GetChatScheduledTaskForUID(ctx, taskID, uid.String()); err != nil {
		return err
	}
	cancelled := string(schema.ChatScheduledTaskStateCancelled)
	if err := store.Database.UpdateChatScheduledTask(ctx, taskID, store.UpdateChatScheduledTaskParams{
		State: &cancelled,
	}); err != nil {
		return err
	}
	if sched := GlobalScheduler(); sched != nil {
		sched.UnregisterTask(taskID)
	}
	return nil
}

// PatchScheduledTaskForUID updates owned task fields from an HTTP request.
func PatchScheduledTaskForUID(ctx context.Context, uid types.Uid, taskID string, req UpdateScheduledTaskRequest) (*ScheduledTaskView, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	task, err := store.Database.GetChatScheduledTaskForUID(ctx, taskID, uid.String())
	if err != nil {
		return nil, err
	}
	if !EditableScheduledTaskState(task.State) {
		return nil, types.Errorf(types.ErrForbidden, "task is not editable")
	}

	params, err := buildPatchScheduleParams(task, req)
	if err != nil {
		return nil, err
	}
	updated, err := applyScheduledTaskUpdate(ctx, taskID, params)
	if err != nil {
		return nil, err
	}
	view := taskViewFromRow(updated)
	return &view, nil
}

// ListScheduledTaskRuns returns recent runs for one owned task.
func ListScheduledTaskRuns(ctx context.Context, uid types.Uid, taskID string, limit int) ([]ScheduledTaskRunView, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	if _, err := store.Database.GetChatScheduledTaskForUID(ctx, taskID, uid.String()); err != nil {
		return nil, err
	}
	rows, err := store.Database.ListChatScheduledTaskRuns(ctx, taskID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ScheduledTaskRunView, 0, len(rows))
	for _, row := range rows {
		out = append(out, ScheduledTaskRunView{
			RunID:        row.Flag,
			TaskID:       row.TaskID,
			RunSessionID: row.RunSessionID,
			State:        row.State,
			Reply:        row.Reply,
			Error:        row.Error,
			StartedAt:    row.StartedAt,
			FinishedAt:   row.FinishedAt,
		})
	}
	return out, nil
}

func taskViewFromRow(row *gen.ChatScheduledTask) ScheduledTaskView {
	if row == nil {
		return ScheduledTaskView{}
	}
	return ScheduledTaskView{
		TaskID:          row.Flag,
		Name:            row.Name,
		ScheduleKind:    row.ScheduleKind,
		Cron:            row.Cron,
		RunAt:           row.RunAt,
		Prompt:          row.Prompt,
		State:           row.State,
		SourceSessionID: row.SourceSessionID,
		LastRunAt:       row.LastRunAt,
		NextRunAt:       row.NextRunAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

// ParseCreateScheduledTaskRequest validates create request fields.
func ParseCreateScheduledTaskRequest(req CreateScheduledTaskRequest) error {
	parsed := parseCreateScheduleArgs(map[string]any{
		"name":   strings.TrimSpace(req.Name),
		"prompt": strings.TrimSpace(req.Prompt),
		"cron":   strings.TrimSpace(req.Cron),
		"run_at": strings.TrimSpace(req.RunAt),
	})
	if parsed.errText != "" {
		return fmt.Errorf("%s", parsed.errText)
	}
	return nil
}
