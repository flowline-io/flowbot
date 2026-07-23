package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

const maxAgentKnowledgeContentBytes = 65536

var agentKnowledgeWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-knowledge", agentKnowledgePage, route.WithNotAuth()),
	webservice.Get("/agent-knowledge/list", agentKnowledgeTable, route.WithNotAuth()),
	webservice.Get("/agent-knowledge/new", agentKnowledgeNewForm, route.WithNotAuth()),
	webservice.Post("/agent-knowledge", agentKnowledgeCreate, route.WithNotAuth()),
	webservice.Get("/agent-knowledge/:id/edit", agentKnowledgeEditForm, route.WithNotAuth()),
	webservice.Put("/agent-knowledge/:id", agentKnowledgeUpdate, route.WithNotAuth()),
	webservice.Delete("/agent-knowledge/:id", agentKnowledgeDelete, route.WithNotAuth()),
}

func agentKnowledgePage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	q := strings.TrimSpace(ctx.Query("q"))
	items, err := listAgentKnowledgeModels(ctx.Context(), q)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list agent knowledge: %v", err)
	}
	ctx.Type("html")
	return pages.AgentKnowledgePage(items, q).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentKnowledgeTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	q := strings.TrimSpace(ctx.Query("q"))
	items, err := listAgentKnowledgeModels(ctx.Context(), q)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent knowledge")
	}
	ctx.Type("html")
	return partials.AgentKnowledgeTable(items, q).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentKnowledgeNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	q := strings.TrimSpace(ctx.Query("q"))
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-knowledge-form-new" hx-swap-oob="delete"></tr><tr id="agent-knowledge-empty" hx-swap-oob="delete"></tr>`))
	return partials.AgentKnowledgeForm(model.AgentKnowledge{Tags: []string{}}, true, nil, q).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentKnowledgeCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	reqCtx := ctx.Context()
	input := parseAgentKnowledgeForm(ctx)
	errs := validateAgentKnowledgeForm(input)
	if len(errs) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.AgentKnowledgeForm(input, true, errs, "").Render(reqCtx, ctx.Response().BodyWriter())
	}
	now := time.Now().UTC()
	row := &gen.AgentKnowledge{
		Path:      input.Path,
		Title:     input.Title,
		Tags:      input.Tags,
		Summary:   input.Summary,
		Content:   input.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Database.CreateAgentKnowledge(reqCtx, row); err != nil {
		if fieldErrs := mapAgentKnowledgeUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			return partials.AgentKnowledgeForm(input, true, fieldErrs, "").Render(reqCtx, ctx.Response().BodyWriter())
		}
		return toastError(ctx, "Failed to create knowledge document")
	}
	flog.Info("[web] agent knowledge created uid=%s id=%d path=%s", getUID(ctx), row.ID, row.Path)
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="agent-knowledge-empty" hx-swap-oob="delete"></tr>`))
	return partials.AgentKnowledgeRow(agentKnowledgeModelFromGen(row)).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentKnowledgeEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := decodeAgentKnowledgeID(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	item, err := loadAgentKnowledgeModel(reqCtx, id)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Knowledge document not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load knowledge document")
	}
	ctx.Type("html")
	return partials.AgentKnowledgeForm(item, false, nil, "").Render(reqCtx, ctx.Response().BodyWriter())
}

func agentKnowledgeUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := decodeAgentKnowledgeID(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	existing, err := store.Database.GetAgentKnowledgeByID(reqCtx, id)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Knowledge document not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load knowledge document")
	}
	input := parseAgentKnowledgeForm(ctx)
	input.ID = id
	errs := validateAgentKnowledgeForm(input)
	if len(errs) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.AgentKnowledgeForm(input, false, errs, "").Render(reqCtx, ctx.Response().BodyWriter())
	}
	existing.Path = input.Path
	existing.Title = input.Title
	existing.Tags = input.Tags
	existing.Summary = input.Summary
	existing.Content = input.Content
	if err := store.Database.UpdateAgentKnowledge(reqCtx, existing); err != nil {
		if fieldErrs := mapAgentKnowledgeUniqueError(err); len(fieldErrs) > 0 {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			return partials.AgentKnowledgeForm(input, false, fieldErrs, "").Render(reqCtx, ctx.Response().BodyWriter())
		}
		return toastError(ctx, "Failed to update knowledge document")
	}
	flog.Info("[web] agent knowledge updated uid=%s id=%d path=%s", getUID(ctx), id, existing.Path)
	updated, err := loadAgentKnowledgeModel(reqCtx, id)
	if err != nil {
		return toastError(ctx, "Failed to load updated knowledge document")
	}
	ctx.Type("html")
	return partials.AgentKnowledgeRow(updated).Render(reqCtx, ctx.Response().BodyWriter())
}

func agentKnowledgeDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := decodeAgentKnowledgeID(ctx)
	if err != nil {
		return err
	}
	reqCtx := ctx.Context()
	if err := store.Database.DeleteAgentKnowledge(reqCtx, id); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(ctx, "Knowledge document not found")
		}
		return toastError(ctx, "Failed to delete knowledge document")
	}
	flog.Info("[web] agent knowledge deleted uid=%s id=%d", getUID(ctx), id)
	items, err := store.Database.ListAgentKnowledge(reqCtx, store.AgentKnowledgeListFilter{})
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		_ = partials.WriteTableEmptyOOB(
			reqCtx,
			ctx.Response().BodyWriter(),
			"agent-knowledge-empty",
			"#agent-knowledge-rows",
			"6",
			partials.EmptyStateHXCTA(
				"No knowledge documents yet",
				"Add markdown docs for the agent to search and read.",
				"/service/web/agent-knowledge/new",
				"#agent-knowledge-rows",
				"afterbegin",
				"Create document",
			),
		)
	}
	return ctx.SendString("")
}

func decodeAgentKnowledgeID(ctx fiber.Ctx) (int64, error) {
	raw := strings.TrimSpace(ctx.Params("id"))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, types.Errorf(types.ErrInvalidArgument, "invalid knowledge document id")
	}
	return id, nil
}

func parseAgentKnowledgeForm(ctx fiber.Ctx) model.AgentKnowledge {
	return model.AgentKnowledge{
		Path:    strings.TrimSpace(ctx.FormValue("path")),
		Title:   strings.TrimSpace(ctx.FormValue("title")),
		Tags:    parseAgentKnowledgeTags(ctx.FormValue("tags")),
		Summary: strings.TrimSpace(ctx.FormValue("summary")),
		Content: ctx.FormValue("content"),
	}
}

func parseAgentKnowledgeTags(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func validateAgentKnowledgeForm(input model.AgentKnowledge) map[string]string {
	errs := map[string]string{}
	if err := chatagent.ValidateKnowledgePath(input.Path); err != nil {
		errs["path"] = err.Error()
	}
	if strings.TrimSpace(input.Title) == "" {
		errs["title"] = "title is required"
	}
	if strings.TrimSpace(input.Content) == "" {
		errs["content"] = "content is required"
	} else if len(input.Content) > maxAgentKnowledgeContentBytes {
		errs["content"] = "content exceeds maximum size"
	}
	return errs
}

func mapAgentKnowledgeUniqueError(err error) map[string]string {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") {
		return map[string]string{"path": "path already exists"}
	}
	return nil
}

func listAgentKnowledgeModels(ctx context.Context, q string) ([]model.AgentKnowledge, error) {
	rows, err := store.Database.ListAgentKnowledge(ctx, store.AgentKnowledgeListFilter{Q: q})
	if err != nil {
		return nil, err
	}
	items := make([]model.AgentKnowledge, 0, len(rows))
	for _, row := range rows {
		items = append(items, agentKnowledgeModelFromGen(row))
	}
	return items, nil
}

func loadAgentKnowledgeModel(ctx context.Context, id int64) (model.AgentKnowledge, error) {
	row, err := store.Database.GetAgentKnowledgeByID(ctx, id)
	if err != nil {
		return model.AgentKnowledge{}, err
	}
	return agentKnowledgeModelFromGen(row), nil
}

func agentKnowledgeModelFromGen(row *gen.AgentKnowledge) model.AgentKnowledge {
	if row == nil {
		return model.AgentKnowledge{}
	}
	tags := row.Tags
	if tags == nil {
		tags = []string{}
	}
	return model.AgentKnowledge{
		ID:        row.ID,
		Path:      row.Path,
		Title:     row.Title,
		Tags:      tags,
		Summary:   row.Summary,
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
