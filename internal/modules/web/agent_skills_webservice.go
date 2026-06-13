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

var agentSkillSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var agentSkillsWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-skills", agentSkillsPage, route.WithNotAuth()),
	webservice.Get("/agent-skills/list", agentSkillsTable, route.WithNotAuth()),
	webservice.Get("/agent-skills/new", agentSkillNewForm, route.WithNotAuth()),
	webservice.Post("/agent-skills", agentSkillCreate, route.WithNotAuth()),
	webservice.Get("/agent-skills/:flag/edit", agentSkillEditForm, route.WithNotAuth()),
	webservice.Put("/agent-skills/:flag", agentSkillUpdate, route.WithNotAuth()),
	webservice.Delete("/agent-skills/:flag", agentSkillDelete, route.WithNotAuth()),
}

func agentSkillsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := listAgentSkillModels(ctx.Context())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list agent skills: %v", err)
	}
	ctx.Type("html")
	return pages.AgentSkillsPage(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSkillsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := listAgentSkillModels(ctx.Context())
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent skills")
	}
	ctx.Type("html")
	return partials.AgentSkillTable(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSkillNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-skill-form-new" hx-swap-oob="delete"></tr><tr id="agent-skills-empty" hx-swap-oob="delete"></tr>`))
	return partials.AgentSkillForm(model.AgentSkill{Source: "global", Enabled: true}, true, nil).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSkillCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	reqCtx := ctx.Context()
	input := parseAgentSkillForm(ctx)
	errs := validateAgentSkillForm(input, true)
	if len(errs) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.AgentSkillForm(input, true, errs).Render(reqCtx, ctx.Response().BodyWriter())
	}
	now := time.Now().UTC()
	row := &gen.AgentSkill{
		Flag:                   input.Flag,
		Name:                   input.Name,
		Description:            input.Description,
		Content:                input.Content,
		BaseDir:                input.BaseDir,
		Source:                 defaultAgentSkillSource(input.Source),
		Enabled:                input.Enabled,
		DisableModelInvocation: input.DisableModelInvocation,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := store.Database.CreateAgentSkill(reqCtx, row); err != nil {
		if fieldErrs := mapAgentSkillUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			return partials.AgentSkillForm(input, true, fieldErrs).Render(reqCtx, ctx.Response().BodyWriter())
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to create agent skill")
	}
	chatagent.InvalidatePromptCache()
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-skills-empty" hx-swap-oob="delete"></tr>`))
	return partials.AgentSkillRow(agentSkillFromInput(input, now, now)).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentSkillEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeAgentSkillFlag(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	item, err := loadAgentSkillModel(reqCtx, flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Agent skill not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent skill")
	}
	ctx.Type("html")
	return partials.AgentSkillForm(item, false, nil).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentSkillUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeAgentSkillFlag(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	existing, err := store.Database.GetAgentSkillByFlag(reqCtx, flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Agent skill not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent skill")
	}
	input := parseAgentSkillForm(ctx)
	input.Flag = flag
	errs := validateAgentSkillForm(input, false)
	if len(errs) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.AgentSkillForm(input, false, errs).Render(reqCtx, ctx.Response().BodyWriter())
	}
	row := &gen.AgentSkill{
		Flag:                   flag,
		Name:                   input.Name,
		Description:            input.Description,
		Content:                input.Content,
		BaseDir:                input.BaseDir,
		Source:                 defaultAgentSkillSource(input.Source),
		Enabled:                input.Enabled,
		DisableModelInvocation: input.DisableModelInvocation,
		CreatedAt:              existing.CreatedAt,
	}
	if err := store.Database.UpdateAgentSkill(reqCtx, row); err != nil {
		if fieldErrs := mapAgentSkillUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			return partials.AgentSkillForm(input, false, fieldErrs).Render(reqCtx, ctx.Response().BodyWriter())
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to update agent skill")
	}
	chatagent.InvalidatePromptCache()
	updated, err := loadAgentSkillModel(reqCtx, flag)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load updated agent skill")
	}
	ctx.Type("html")
	return partials.AgentSkillRow(updated).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentSkillDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeAgentSkillFlag(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	if err := store.Database.DeleteAgentSkill(reqCtx, flag); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Agent skill not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to delete agent skill")
	}
	chatagent.InvalidatePromptCache()
	items, err := store.Database.ListAgentSkills(reqCtx, false)
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-skills-empty" hx-swap-oob="innerHTML:#agent-skills-rows"><td colspan="6" class="text-center text-base-content/50">No agent skills found.</td></tr>`))
	}
	return ctx.SendString("")
}

func decodeAgentSkillFlag(ctx fiber.Ctx) (string, error) {
	flag, err := url.PathUnescape(ctx.Params("flag"))
	if err != nil || strings.TrimSpace(flag) == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "invalid agent skill flag")
	}
	return flag, nil
}

func parseAgentSkillForm(ctx fiber.Ctx) model.AgentSkill {
	return model.AgentSkill{
		Flag:                   strings.TrimSpace(ctx.FormValue("flag")),
		Name:                   strings.TrimSpace(ctx.FormValue("name")),
		Description:            strings.TrimSpace(ctx.FormValue("description")),
		Content:                ctx.FormValue("content"),
		BaseDir:                strings.TrimSpace(ctx.FormValue("base_dir")),
		Source:                 strings.TrimSpace(ctx.FormValue("source")),
		Enabled:                ctx.FormValue("enabled") == "true",
		DisableModelInvocation: ctx.FormValue("disable_model_invocation") == "true",
	}
}

func validateAgentSkillForm(item model.AgentSkill, isNew bool) map[string]string {
	errs := make(map[string]string)
	if isNew {
		if item.Flag == "" {
			errs["flag"] = "Flag is required"
		} else if !agentSkillSlugPattern.MatchString(item.Flag) {
			errs["flag"] = "Flag must be lowercase letters, numbers, and hyphens"
		}
	}
	if item.Name == "" {
		errs["name"] = "Name is required"
	} else if !agentSkillSlugPattern.MatchString(item.Name) {
		errs["name"] = "Name must be lowercase letters, numbers, and hyphens"
	}
	if item.Description == "" {
		errs["description"] = "Description is required"
	}
	if strings.TrimSpace(item.Content) == "" {
		errs["content"] = "Content is required"
	}
	return errs
}

func defaultAgentSkillSource(source string) string {
	if source == "" {
		return "global"
	}
	return source
}

func mapAgentSkillUniqueError(err error) map[string]string {
	msg := err.Error()
	errs := make(map[string]string)
	if strings.Contains(msg, "agent_skills_flag_key") {
		errs["flag"] = "Flag already exists"
	}
	if strings.Contains(msg, "agent_skills_name_key") {
		errs["name"] = "Name already exists"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func listAgentSkillModels(ctx context.Context) ([]model.AgentSkill, error) {
	rows, err := store.Database.ListAgentSkills(ctx, false)
	if err != nil {
		return nil, err
	}
	items := make([]model.AgentSkill, 0, len(rows))
	for _, row := range rows {
		items = append(items, agentSkillFromRow(row))
	}
	return items, nil
}

func loadAgentSkillModel(ctx context.Context, flag string) (model.AgentSkill, error) {
	row, err := store.Database.GetAgentSkillByFlag(ctx, flag)
	if err != nil {
		return model.AgentSkill{}, err
	}
	return agentSkillFromRow(row), nil
}

func agentSkillFromRow(row *gen.AgentSkill) model.AgentSkill {
	return model.AgentSkill{
		Flag:                   row.Flag,
		Name:                   row.Name,
		Description:            row.Description,
		Content:                row.Content,
		BaseDir:                row.BaseDir,
		Source:                 row.Source,
		Enabled:                row.Enabled,
		DisableModelInvocation: row.DisableModelInvocation,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func agentSkillFromInput(item model.AgentSkill, createdAt, updatedAt time.Time) model.AgentSkill {
	item.Source = defaultAgentSkillSource(item.Source)
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item
}
