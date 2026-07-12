package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var agentScheduledTasksWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-scheduled-tasks", agentScheduledTasksPage, route.WithNotAuth()),
	webservice.Get("/agent-scheduled-tasks/list", agentScheduledTasksTable, route.WithNotAuth()),
	webservice.Get("/agent-scheduled-tasks/:id", agentScheduledTaskDetailPage, route.WithNotAuth()),
	webservice.Put("/agent-scheduled-tasks/:id/state", agentScheduledTaskSetState, route.WithNotAuth()),
}

func agentScheduledTasksPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return renderError(ctx, "Chat agent is not enabled")
	}
	items, err := listScheduledTaskModels(ctx)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list scheduled tasks: %v", err)
	}
	ctx.Type("html")
	return pages.AgentScheduledTasksPage(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentScheduledTasksTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return renderError(ctx, "Chat agent is not enabled")
	}
	items, err := listScheduledTaskModels(ctx)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load scheduled tasks")
	}
	ctx.Type("html")
	return partials.AgentScheduledTaskTable(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentScheduledTaskDetailPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).SendString("chat agent is not enabled")
	}
	uid, err := webUID(ctx)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).SendString("unauthorized")
	}
	taskID := ctx.Params("id")
	if taskID == "" {
		return ctx.Status(http.StatusBadRequest).SendString("task id required")
	}
	task, err := chatagent.GetScheduledTaskForUID(ctx.Context(), uid, taskID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("scheduled task not found")
		}
		return types.Errorf(types.ErrInternal, "get scheduled task: %v", err)
	}
	runs, err := chatagent.ListScheduledTaskRuns(ctx.Context(), uid, taskID, 20)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("scheduled task not found")
		}
		return types.Errorf(types.ErrInternal, "list scheduled task runs: %v", err)
	}
	ctx.Type("html")
	return pages.AgentScheduledTaskDetailPage(
		mapScheduledTask(*task),
		mapScheduledTaskRuns(runs),
	).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentScheduledTaskSetState(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return renderError(ctx, "Chat agent is not enabled")
	}
	uid, err := webUID(ctx)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).SendString("unauthorized")
	}
	taskID := ctx.Params("id")
	if taskID == "" {
		return ctx.Status(http.StatusBadRequest).SendString("task id required")
	}
	state := strings.TrimSpace(ctx.FormValue("state"))
	if state == "" {
		ctx.Status(http.StatusBadRequest)
		return renderError(ctx, "state is required")
	}
	task, err := chatagent.SetScheduledTaskStateForUID(ctx.Context(), uid, taskID, state)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("scheduled task not found")
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			ctx.Status(http.StatusBadRequest)
			return renderError(ctx, "invalid state")
		}
		return types.Errorf(types.ErrInternal, "set scheduled task state: %v", err)
	}
	ctx.Type("html")
	return partials.AgentScheduledTaskStatePanel(mapScheduledTask(*task)).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func listScheduledTaskModels(ctx fiber.Ctx) ([]model.AgentScheduledTask, error) {
	uid, err := webUID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := chatagent.ListScheduledTasksForUID(ctx.Context(), uid, []string{
		string(schema.ChatScheduledTaskStateActive),
		string(schema.ChatScheduledTaskStatePaused),
		string(schema.ChatScheduledTaskStateCancelled),
		string(schema.ChatScheduledTaskStateCompleted),
		string(schema.ChatScheduledTaskStateFailed),
		string(schema.ChatScheduledTaskStateMissed),
	})
	if err != nil {
		return nil, err
	}
	return mapScheduledTasks(items), nil
}

func mapScheduledTasks(items []chatagent.ScheduledTaskView) []model.AgentScheduledTask {
	out := make([]model.AgentScheduledTask, 0, len(items))
	for _, item := range items {
		out = append(out, mapScheduledTask(item))
	}
	return out
}

func mapScheduledTask(item chatagent.ScheduledTaskView) model.AgentScheduledTask {
	return model.AgentScheduledTask{
		TaskID:          item.TaskID,
		Name:            item.Name,
		ScheduleKind:    item.ScheduleKind,
		Cron:            item.Cron,
		RunAt:           item.RunAt,
		Prompt:          item.Prompt,
		State:           item.State,
		SourceSessionID: item.SourceSessionID,
		LastRunAt:       item.LastRunAt,
		NextRunAt:       item.NextRunAt,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func mapScheduledTaskRuns(items []chatagent.ScheduledTaskRunView) []model.AgentScheduledTaskRun {
	out := make([]model.AgentScheduledTaskRun, 0, len(items))
	for _, item := range items {
		out = append(out, model.AgentScheduledTaskRun{
			RunID:        item.RunID,
			TaskID:       item.TaskID,
			RunSessionID: item.RunSessionID,
			State:        item.State,
			Reply:        item.Reply,
			Error:        item.Error,
			StartedAt:    item.StartedAt,
			FinishedAt:   item.FinishedAt,
		})
	}
	return out
}
