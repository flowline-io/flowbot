package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cronutil"
	"github.com/flowline-io/flowbot/pkg/types"
)

type createScheduleArgs struct {
	name    string
	prompt  string
	kind    string
	cron    string
	runAt   *time.Time
	errText string
}

func parseCreateScheduleArgs(args map[string]any) createScheduleArgs {
	out := createScheduleArgs{
		name:   stringArg(args, "name"),
		prompt: stringArg(args, "prompt"),
		cron:   stringArg(args, "cron"),
	}
	runAtRaw := stringArg(args, "run_at")
	if out.name == "" {
		out.errText = "name is required"
		return out
	}
	if out.prompt == "" {
		out.errText = "prompt is required"
		return out
	}
	if out.cron == "" && runAtRaw == "" {
		out.errText = "provide either cron or run_at"
		return out
	}
	if out.cron != "" && runAtRaw != "" {
		out.errText = errCronRunAtConflict.Error()
		return out
	}
	out.kind = string(schema.ChatScheduledTaskKindCron)
	if runAtRaw != "" {
		out.kind = string(schema.ChatScheduledTaskKindOnce)
		parsed, err := ParseRunAt(runAtRaw)
		if err != nil {
			out.errText = err.Error()
			return out
		}
		out.runAt = &parsed
	}
	if err := ValidateScheduleInput(out.kind, out.cron, out.runAt); err != nil {
		out.errText = err.Error()
	}
	return out
}

func newScheduledTaskRecord(ctx context.Context, deps ScheduleToolDeps, parsed createScheduleArgs) (*gen.ChatScheduledTask, error) {
	taskID := types.Id()
	now := time.Now().UTC()
	delivery := ResolveDeliveryContext(ctx, deps.SessionID)
	task := &gen.ChatScheduledTask{
		Flag:            taskID,
		UID:             deps.UID.String(),
		Name:            parsed.name,
		ScheduleKind:    parsed.kind,
		Cron:            parsed.cron,
		RunAt:           parsed.runAt,
		Prompt:          parsed.prompt,
		Delivery:        deliveryToMap(delivery),
		SourceSessionID: deps.SessionID,
		State:           string(schema.ChatScheduledTaskStateActive),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if parsed.kind == string(schema.ChatScheduledTaskKindOnce) && parsed.runAt != nil {
		task.NextRunAt = parsed.runAt
	}
	if parsed.kind == string(schema.ChatScheduledTaskKindCron) && parsed.cron != "" {
		next, err := cronutil.NextRun(parsed.cron, now)
		if err != nil {
			return nil, err
		}
		task.NextRunAt = &next
	}
	return task, nil
}

func persistScheduledTask(ctx context.Context, deps ScheduleToolDeps, parsed createScheduleArgs) (*gen.ChatScheduledTask, error) {
	task, err := newScheduledTaskRecord(ctx, deps, parsed)
	if err != nil {
		return nil, err
	}
	if err := store.Database.CreateChatScheduledTask(ctx, task); err != nil {
		return nil, err
	}
	if err := syncTaskWithScheduler(task); err != nil {
		if derr := store.Database.DeleteChatScheduledTask(ctx, task.Flag); derr != nil {
			return nil, fmt.Errorf("register task: %w (rollback failed: %v)", err, derr)
		}
		return nil, fmt.Errorf("register task: %w", err)
	}
	return task, nil
}

type updateScheduleArgs struct {
	taskID  string
	name    string
	prompt  string
	cron    string
	runAt   string
	state   string
	errText string
}

func parseUpdateScheduleArgs(args map[string]any) updateScheduleArgs {
	out := updateScheduleArgs{
		taskID: stringArg(args, "task_id"),
		name:   stringArg(args, "name"),
		prompt: stringArg(args, "prompt"),
		cron:   stringArg(args, "cron"),
		runAt:  stringArg(args, "run_at"),
		state:  strings.TrimSpace(stringArg(args, "state")),
	}
	if out.taskID == "" {
		out.errText = errTaskIDRequired.Error()
		return out
	}
	if out.name == "" && out.prompt == "" && out.cron == "" && out.runAt == "" && out.state == "" {
		out.errText = errNoUpdateFields.Error()
		return out
	}
	if out.cron != "" && out.runAt != "" {
		out.errText = errCronRunAtConflict.Error()
	}
	if out.state != "" && out.state != string(schema.ChatScheduledTaskStateActive) && out.state != string(schema.ChatScheduledTaskStatePaused) {
		out.errText = errInvalidTaskState.Error()
	}
	return out
}

func buildTaskUpdateParams(task *gen.ChatScheduledTask, parsed updateScheduleArgs) (store.UpdateChatScheduledTaskParams, string) {
	params := store.UpdateChatScheduledTaskParams{}
	if parsed.name != "" {
		params.Name = &parsed.name
	}
	if parsed.prompt != "" {
		params.Prompt = &parsed.prompt
	}
	if parsed.cron != "" {
		if task.ScheduleKind != string(schema.ChatScheduledTaskKindCron) {
			return params, errWrongKindCron.Error()
		}
		if err := ValidateScheduleInput(string(schema.ChatScheduledTaskKindCron), parsed.cron, nil); err != nil {
			return params, err.Error()
		}
		params.Cron = &parsed.cron
		if next, err := cronutil.NextRun(parsed.cron, time.Now().UTC()); err == nil {
			params.NextRunAt = &next
		}
	}
	if parsed.runAt != "" {
		if task.ScheduleKind != string(schema.ChatScheduledTaskKindOnce) {
			return params, errWrongKindOnce.Error()
		}
		parsedTime, err := ParseRunAt(parsed.runAt)
		if err != nil {
			return params, err.Error()
		}
		if err := ValidateScheduleInput(string(schema.ChatScheduledTaskKindOnce), "", &parsedTime); err != nil {
			return params, err.Error()
		}
		params.RunAt = &parsedTime
		params.NextRunAt = &parsedTime
	}
	if parsed.state != "" {
		params.State = &parsed.state
	}
	return params, ""
}

func buildPatchScheduleParams(task *gen.ChatScheduledTask, req UpdateScheduledTaskRequest) (store.UpdateChatScheduledTaskParams, error) {
	params := store.UpdateChatScheduledTaskParams{}
	hasField := false
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return params, types.Errorf(types.ErrInvalidArgument, "name cannot be empty")
		}
		params.Name = &name
		hasField = true
	}
	if req.Prompt != nil {
		prompt := strings.TrimSpace(*req.Prompt)
		if prompt == "" {
			return params, types.Errorf(types.ErrInvalidArgument, "prompt cannot be empty")
		}
		params.Prompt = &prompt
		hasField = true
	}
	if req.Cron != nil {
		patch, err := patchCronField(task, strings.TrimSpace(*req.Cron))
		if err != nil {
			return params, err
		}
		params.Cron = patch.Cron
		params.NextRunAt = patch.NextRunAt
		hasField = true
	}
	if req.RunAt != nil {
		patch, err := patchRunAtField(task, *req.RunAt)
		if err != nil {
			return params, err
		}
		params.RunAt = patch.RunAt
		params.NextRunAt = patch.NextRunAt
		hasField = true
	}
	if req.State != nil {
		state := strings.TrimSpace(*req.State)
		if state != string(schema.ChatScheduledTaskStateActive) && state != string(schema.ChatScheduledTaskStatePaused) {
			return params, types.Errorf(types.ErrInvalidArgument, "%s", errInvalidTaskState.Error())
		}
		params.State = &state
		hasField = true
	}
	if !hasField {
		return params, types.Errorf(types.ErrInvalidArgument, "%s", errNoUpdateFields.Error())
	}
	return params, nil
}

type schedulePatchFields struct {
	Cron      *string
	RunAt     *time.Time
	NextRunAt *time.Time
}

func patchCronField(task *gen.ChatScheduledTask, cronExpr string) (schedulePatchFields, error) {
	if task.ScheduleKind != string(schema.ChatScheduledTaskKindCron) {
		return schedulePatchFields{}, types.Errorf(types.ErrInvalidArgument, "%s", errWrongKindCron.Error())
	}
	if err := cronutil.ValidateExpr(cronExpr); err != nil {
		return schedulePatchFields{}, types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	out := schedulePatchFields{Cron: &cronExpr}
	if next, err := cronutil.NextRun(cronExpr, time.Now().UTC()); err == nil {
		out.NextRunAt = &next
	}
	return out, nil
}

func patchRunAtField(task *gen.ChatScheduledTask, raw string) (schedulePatchFields, error) {
	if task.ScheduleKind != string(schema.ChatScheduledTaskKindOnce) {
		return schedulePatchFields{}, types.Errorf(types.ErrInvalidArgument, "%s", errWrongKindOnce.Error())
	}
	parsed, err := ParseRunAt(raw)
	if err != nil {
		return schedulePatchFields{}, types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	if err := ValidateScheduleInput(string(schema.ChatScheduledTaskKindOnce), "", &parsed); err != nil {
		return schedulePatchFields{}, types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return schedulePatchFields{RunAt: &parsed, NextRunAt: &parsed}, nil
}

func scheduleDescription(task *gen.ChatScheduledTask) string {
	if task.ScheduleKind == string(schema.ChatScheduledTaskKindOnce) && task.RunAt != nil {
		return task.RunAt.UTC().Format(time.RFC3339)
	}
	return task.Cron
}

func formatScheduledTaskListLine(task *gen.ChatScheduledTask) string {
	return fmt.Sprintf("- id=%s name=%q kind=%s state=%s schedule=%s prompt=%q",
		task.Flag, task.Name, task.ScheduleKind, task.State, scheduleDescription(task), promptSummary(task.Prompt, 80))
}

func applyScheduledTaskUpdate(ctx context.Context, taskID string, params store.UpdateChatScheduledTaskParams) (*gen.ChatScheduledTask, error) {
	if err := store.Database.UpdateChatScheduledTask(ctx, taskID, params); err != nil {
		return nil, err
	}
	updated, err := store.Database.GetChatScheduledTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if err := syncTaskWithScheduler(updated); err != nil {
		return nil, fmt.Errorf("reschedule task: %w", err)
	}
	return updated, nil
}
