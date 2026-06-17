package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
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
	return partials.AgentSubagentForm(model.AgentSubagent{Source: "global", Enabled: true}, true, nil).
		Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return partials.AgentSubagentForm(input, true, errs).Render(reqCtx, ctx.Response().BodyWriter())
	}
	now := time.Now().UTC()
	row := &gen.AgentSubagent{
		Flag:         input.Flag,
		Name:         input.Name,
		Description:  input.Description,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
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
			return partials.AgentSubagentForm(input, true, fieldErrs).Render(reqCtx, ctx.Response().BodyWriter())
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
	return partials.AgentSubagentForm(item, false, nil).Render(reqCtx, ctx.Response().BodyWriter())
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
		return partials.AgentSubagentForm(input, false, errs).Render(reqCtx, ctx.Response().BodyWriter())
	}
	row := &gen.AgentSubagent{
		Flag:         flag,
		Name:         input.Name,
		Description:  input.Description,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
		Model:        input.Model,
		Source:       defaultAgentSubagentSource(input.Source),
		Enabled:      input.Enabled,
		CreatedAt:    existing.CreatedAt,
	}
	if err := store.Database.UpdateAgentSubagent(reqCtx, row); err != nil {
		if fieldErrs := mapAgentSubagentUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			return partials.AgentSubagentForm(input, false, fieldErrs).Render(reqCtx, ctx.Response().BodyWriter())
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

func decodeAgentSubagentFlag(ctx fiber.Ctx) (string, error) {
	flag, err := url.PathUnescape(ctx.Params("flag"))
	if err != nil || strings.TrimSpace(flag) == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "invalid agent subagent flag")
	}
	return flag, nil
}

func parseAgentSubagentForm(ctx fiber.Ctx) model.AgentSubagent {
	return model.AgentSubagent{
		Flag:         strings.TrimSpace(ctx.FormValue("flag")),
		Name:         strings.TrimSpace(ctx.FormValue("name")),
		Description:  strings.TrimSpace(ctx.FormValue("description")),
		SystemPrompt: ctx.FormValue("system_prompt"),
		Tools:        parseAgentSubagentTools(ctx.FormValue("tools")),
		Model:        strings.TrimSpace(ctx.FormValue("model")),
		Source:       strings.TrimSpace(ctx.FormValue("source")),
		Enabled:      ctx.FormValue("enabled") == "true",
	}
}

func parseAgentSubagentTools(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' ' || r == '\t'
	})
	tools := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		tools = append(tools, name)
	}
	return tools
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
