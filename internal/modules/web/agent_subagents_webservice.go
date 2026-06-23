package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var agentSubagentSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var agentSubagentsWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-subagents", agentSubagentsPage, route.WithNotAuth()),
	webservice.Get("/agent-subagents/list", agentSubagentsTable, route.WithNotAuth()),
	webservice.Get("/agent-subagents/new", agentSubagentNewForm, route.WithNotAuth()),
	webservice.Post("/agent-subagents", agentSubagentCreate, route.WithNotAuth()),
	webservice.Get("/agent-subagents/:flag/edit", agentSubagentEditForm, route.WithNotAuth()),
	webservice.Put("/agent-subagents/:flag", agentSubagentUpdate, route.WithNotAuth()),
	webservice.Delete("/agent-subagents/:flag", agentSubagentDelete, route.WithNotAuth()),
	webservice.Get("/agent-subagents/tasks", agentSubagentTasksTable, route.WithNotAuth()),
	webservice.Get("/agent-subagents/tasks/:id", agentSubagentTaskDetail, route.WithNotAuth()),
}

func agentSubagentsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := listAgentSubagentModels(ctx.Context())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list agent subagents: %v", err)
	}
	ctx.Type("html")
	return pages.AgentSubagentsPage(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSubagentsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := listAgentSubagentModels(ctx.Context())
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent subagents")
	}
	ctx.Type("html")
	return partials.AgentSubagentTable(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSubagentNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-subagent-form-new" hx-swap-oob="delete"></tr><tr id="agent-subagents-empty" hx-swap-oob="delete"></tr>`))
	params, err := buildAgentSubagentFormParams(ctx.Context(), model.AgentSubagent{Source: "global", Enabled: true}, true, nil)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load subagent form options")
	}
	return partials.AgentSubagentForm(params).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSubagentCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	reqCtx := ctx.Context()
	input := parseAgentSubagentForm(ctx)
	errs := validateAgentSubagentForm(input, true)
	if len(errs) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		params, buildErr := buildAgentSubagentFormParams(reqCtx, input, true, errs)
		if buildErr != nil {
			return renderError(ctx, "Failed to load subagent form options")
		}
		return partials.AgentSubagentForm(params).Render(reqCtx, ctx.Response().BodyWriter())
	}
	now := time.Now().UTC()
	row := &gen.AgentSubagent{
		Flag:         input.Flag,
		Name:         input.Name,
		Description:  input.Description,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
		Skills:       input.Skills,
		Model:        input.Model,
		Source:       defaultAgentSubagentSource(input.Source),
		Enabled:      input.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Database.CreateAgentSubagent(reqCtx, row); err != nil {
		if fieldErrs := mapAgentSubagentUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			params, buildErr := buildAgentSubagentFormParams(reqCtx, input, true, fieldErrs)
			if buildErr != nil {
				return renderError(ctx, "Failed to load subagent form options")
			}
			return partials.AgentSubagentForm(params).Render(reqCtx, ctx.Response().BodyWriter())
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to create agent subagent")
	}
	chatagent.InvalidatePromptCache()
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-subagents-empty" hx-swap-oob="delete"></tr>`))
	return partials.AgentSubagentRow(agentSubagentFromInput(input, now, now)).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentSubagentEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeAgentSubagentFlag(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	item, err := loadAgentSubagentModel(reqCtx, flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Agent subagent not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent subagent")
	}
	ctx.Type("html")
	params, err := buildAgentSubagentFormParams(reqCtx, item, false, nil)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load subagent form options")
	}
	return partials.AgentSubagentForm(params).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentSubagentUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeAgentSubagentFlag(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	existing, err := store.Database.GetAgentSubagentByFlag(reqCtx, flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Agent subagent not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent subagent")
	}
	input := parseAgentSubagentForm(ctx)
	input.Flag = flag
	errs := validateAgentSubagentForm(input, false)
	if len(errs) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		params, buildErr := buildAgentSubagentFormParams(reqCtx, input, false, errs)
		if buildErr != nil {
			return renderError(ctx, "Failed to load subagent form options")
		}
		return partials.AgentSubagentForm(params).Render(reqCtx, ctx.Response().BodyWriter())
	}
	row := &gen.AgentSubagent{
		Flag:         flag,
		Name:         input.Name,
		Description:  input.Description,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
		Skills:       input.Skills,
		Model:        input.Model,
		Source:       defaultAgentSubagentSource(input.Source),
		Enabled:      input.Enabled,
		CreatedAt:    existing.CreatedAt,
	}
	if err := store.Database.UpdateAgentSubagent(reqCtx, row); err != nil {
		if fieldErrs := mapAgentSubagentUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			params, buildErr := buildAgentSubagentFormParams(reqCtx, input, false, fieldErrs)
			if buildErr != nil {
				return renderError(ctx, "Failed to load subagent form options")
			}
			return partials.AgentSubagentForm(params).Render(reqCtx, ctx.Response().BodyWriter())
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to update agent subagent")
	}
	chatagent.InvalidatePromptCache()
	updated, err := loadAgentSubagentModel(reqCtx, flag)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load updated agent subagent")
	}
	ctx.Type("html")
	return partials.AgentSubagentRow(updated).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentSubagentDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeAgentSubagentFlag(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	if err := store.Database.DeleteAgentSubagent(reqCtx, flag); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Agent subagent not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to delete agent subagent")
	}
	chatagent.InvalidatePromptCache()
	items, err := store.Database.ListAgentSubagents(reqCtx, false)
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-subagents-empty" hx-swap-oob="innerHTML:#agent-subagents-rows"><td colspan="7" class="text-center text-base-content/50">No agent subagents found.</td></tr>`))
	}
	return ctx.SendString("")
}

func agentSubagentTasksTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := listAgentSubagentTaskModels(ctx.Context(), "", 100)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load subagent tasks")
	}
	ctx.Type("html")
	return partials.AgentSubagentTaskTable(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSubagentTaskDetail(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := decodeAgentSubagentTaskID(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	item, err := loadAgentSubagentTaskModel(reqCtx, id)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Subagent task not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load subagent task")
	}
	ctx.Type("html")
	return partials.AgentSubagentTaskDetail(item).Render(reqCtx, ctx.Response().BodyWriter())
}

func decodeAgentSubagentTaskID(ctx fiber.Ctx) (int64, error) {
	raw := strings.TrimSpace(ctx.Params("id"))
	if raw == "" {
		return 0, types.Errorf(types.ErrInvalidArgument, "invalid subagent task id")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, types.Errorf(types.ErrInvalidArgument, "invalid subagent task id")
	}
	return id, nil
}

func decodeAgentSubagentFlag(ctx fiber.Ctx) (string, error) {
	flag, err := url.PathUnescape(ctx.Params("flag"))
	if err != nil || strings.TrimSpace(flag) == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "invalid agent subagent flag")
	}
	return flag, nil
}

func parseAgentSubagentForm(ctx fiber.Ctx) model.AgentSubagent {
	args := ctx.RequestCtx().PostArgs()
	return model.AgentSubagent{
		Flag:         strings.TrimSpace(ctx.FormValue("flag")),
		Name:         strings.TrimSpace(ctx.FormValue("name")),
		Description:  strings.TrimSpace(ctx.FormValue("description")),
		SystemPrompt: ctx.FormValue("system_prompt"),
		Tools:        parseAgentSubagentMultiValues(args.PeekMulti("tools")),
		Skills:       parseAgentSubagentMultiValues(args.PeekMulti("skills")),
		Model:        strings.TrimSpace(ctx.FormValue("model")),
		Source:       strings.TrimSpace(ctx.FormValue("source")),
		Enabled:      ctx.FormValue("enabled") == "true",
	}
}

func parseAgentSubagentMultiValues(values [][]byte) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		name := strings.TrimSpace(string(raw))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		items = append(items, name)
	}
	return items
}

func buildAgentSubagentFormParams(ctx context.Context, item model.AgentSubagent, isNew bool, errs map[string]string) (model.AgentSubagentFormParams, error) {
	skills, err := listAgentSubagentSkillOptions(ctx)
	if err != nil {
		return model.AgentSubagentFormParams{}, err
	}
	return model.AgentSubagentFormParams{
		Item:            item,
		IsNew:           isNew,
		Errors:          errs,
		AvailableTools:  chatagent.SelectableSubagentTools(),
		AvailableSkills: skills,
	}, nil
}

func listAgentSubagentSkillOptions(ctx context.Context) ([]model.AgentSubagentSkillOption, error) {
	rows, err := store.Database.ListAgentSkills(ctx, true)
	if err != nil {
		return nil, err
	}
	options := make([]model.AgentSubagentSkillOption, 0, len(rows))
	for _, row := range rows {
		if row.DisableModelInvocation {
			continue
		}
		options = append(options, model.AgentSubagentSkillOption{
			Name:        row.Name,
			Description: row.Description,
		})
	}
	return options, nil
}

func validateAgentSubagentForm(item model.AgentSubagent, isNew bool) map[string]string {
	errs := make(map[string]string)
	if isNew {
		if item.Flag == "" {
			errs["flag"] = "Flag is required"
		} else if !agentSubagentSlugPattern.MatchString(item.Flag) {
			errs["flag"] = "Flag must be lowercase letters, numbers, and hyphens"
		}
	}
	if item.Name == "" {
		errs["name"] = "Name is required"
	} else if !agentSubagentSlugPattern.MatchString(item.Name) {
		errs["name"] = "Name must be lowercase letters, numbers, and hyphens"
	}
	if item.Description == "" {
		errs["description"] = "Description is required"
	}
	if strings.TrimSpace(item.SystemPrompt) == "" {
		errs["system_prompt"] = "System prompt is required"
	}
	return errs
}

func defaultAgentSubagentSource(source string) string {
	if source == "" {
		return "global"
	}
	return source
}

func mapAgentSubagentUniqueError(err error) map[string]string {
	msg := err.Error()
	errs := make(map[string]string)
	if strings.Contains(msg, "agent_subagents_flag_key") {
		errs["flag"] = "Flag already exists"
	}
	if strings.Contains(msg, "agent_subagents_name_key") {
		errs["name"] = "Name already exists"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func listAgentSubagentModels(ctx context.Context) ([]model.AgentSubagent, error) {
	rows, err := store.Database.ListAgentSubagents(ctx, false)
	if err != nil {
		return nil, err
	}
	items := make([]model.AgentSubagent, 0, len(rows))
	for _, row := range rows {
		items = append(items, agentSubagentFromRow(row))
	}
	return items, nil
}

func loadAgentSubagentModel(ctx context.Context, flag string) (model.AgentSubagent, error) {
	row, err := store.Database.GetAgentSubagentByFlag(ctx, flag)
	if err != nil {
		return model.AgentSubagent{}, err
	}
	return agentSubagentFromRow(row), nil
}

func agentSubagentFromRow(row *gen.AgentSubagent) model.AgentSubagent {
	return model.AgentSubagent{
		Flag:         row.Flag,
		Name:         row.Name,
		Description:  row.Description,
		SystemPrompt: row.SystemPrompt,
		Tools:        append([]string(nil), row.Tools...),
		Skills:       append([]string(nil), row.Skills...),
		Model:        row.Model,
		Source:       row.Source,
		Enabled:      row.Enabled,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func agentSubagentFromInput(item model.AgentSubagent, createdAt, updatedAt time.Time) model.AgentSubagent {
	item.Source = defaultAgentSubagentSource(item.Source)
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item
}

func listAgentSubagentTaskModels(ctx context.Context, sessionID string, limit int) ([]model.AgentSubagentTask, error) {
	rows, err := store.Database.ListAgentSubagentTasks(ctx, sessionID, limit)
	if err != nil {
		return nil, err
	}
	items := make([]model.AgentSubagentTask, 0, len(rows))
	for _, row := range rows {
		items = append(items, agentSubagentTaskFromRow(row))
	}
	return items, nil
}

func loadAgentSubagentTaskModel(ctx context.Context, id int64) (model.AgentSubagentTask, error) {
	row, err := store.Database.GetAgentSubagentTask(ctx, id)
	if err != nil {
		return model.AgentSubagentTask{}, err
	}
	return agentSubagentTaskFromRow(row), nil
}

func agentSubagentTaskFromRow(row *gen.AgentSubagentTask) model.AgentSubagentTask {
	return model.AgentSubagentTask{
		ID:           row.ID,
		SessionID:    row.SessionID,
		SubagentName: row.SubagentName,
		Description:  row.Description,
		Prompt:       row.Prompt,
		Status:       row.Status,
		Result:       row.Result,
		ErrorText:    row.ErrorText,
		Depth:        row.Depth,
		StartedAt:    row.StartedAt,
		FinishedAt:   row.FinishedAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
