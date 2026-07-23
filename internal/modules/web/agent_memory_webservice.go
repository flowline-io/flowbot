package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var agentMemoryWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-memory", agentMemoryPage, route.WithNotAuth()),
	webservice.Get("/agent-memory/list", agentMemoryTable, route.WithNotAuth()),
	webservice.Get("/agent-memory/facts", agentMemoryListFacts, route.WithNotAuth()),
	webservice.Put("/agent-memory/facts", agentMemoryUpsertFact, route.WithNotAuth()),
	webservice.Post("/agent-memory/facts/save", agentMemorySaveFactForm, route.WithNotAuth()),
	webservice.Post("/agent-memory/facts/delete", agentMemoryDeleteFactForm, route.WithNotAuth()),
	webservice.Delete("/agent-memory/facts", agentMemoryDeleteFact, route.WithNotAuth()),
}

type agentMemoryFactRequest struct {
	Scope  string `json:"scope"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Pinned bool   `json:"pinned"`
}

func agentMemoryPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := agentMemoryScopeQuery(ctx)
	items, err := listAgentMemoryFactModels(ctx.Context(), scope)
	if err != nil {
		return types.WrapError(types.ErrInternal, "list memory facts", err)
	}
	ctx.Type("html")
	return pages.AgentMemoryPage(items, scope).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentMemoryTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := agentMemoryScopeQuery(ctx)
	items, err := listAgentMemoryFactModels(ctx.Context(), scope)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load memory facts")
	}
	ctx.Type("html")
	return partials.AgentMemoryTable(items, scope).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentMemoryListFacts(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := strings.TrimSpace(ctx.Query("scope"))
	if scope == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("scope is required")))
	}
	items, err := listAgentMemoryFactModels(ctx.Context(), scope)
	if err != nil {
		return types.WrapError(types.ErrInternal, "list memory facts", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(items))
}

func agentMemoryUpsertFact(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	var req agentMemoryFactRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("invalid JSON body")))
	}
	scope := strings.TrimSpace(req.Scope)
	key := strings.TrimSpace(req.Key)
	value := strings.TrimSpace(req.Value)
	if scope == "" || key == "" || value == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("scope, key, and value are required")))
	}
	row, err := updateExistingAgentMemoryFact(ctx.Context(), scope, key, value, req.Pinned)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(protocol.NewFailedResponse(errors.New("fact not found; create facts with memory_set")))
		}
		return types.WrapError(types.ErrInvalidArgument, "update memory fact", err)
	}
	chatagent.InvalidatePromptCache()
	return ctx.JSON(protocol.NewSuccessResponse(agentMemoryFactModelFromGen(row)))
}

func agentMemorySaveFactForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := strings.TrimSpace(ctx.FormValue("scope"))
	key := strings.TrimSpace(ctx.FormValue("key"))
	value := strings.TrimSpace(ctx.FormValue("value"))
	pinned := ctx.FormValue("pinned") == "on" || ctx.FormValue("pinned") == "true" || ctx.FormValue("pinned") == "1"
	if scope == "" {
		scope = "default"
	}
	if key == "" || value == "" {
		return toastError(ctx, "Key and value are required")
	}
	if _, err := updateExistingAgentMemoryFact(ctx.Context(), scope, key, value, pinned); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(ctx, "Fact not found; create facts with memory_set")
		}
		return toastError(ctx, "Failed to save fact")
	}
	chatagent.InvalidatePromptCache()
	return renderAgentMemoryTable(ctx, scope)
}

func agentMemoryDeleteFactForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := strings.TrimSpace(ctx.FormValue("scope"))
	key := strings.TrimSpace(ctx.FormValue("key"))
	if scope == "" {
		scope = "default"
	}
	if key == "" {
		return toastError(ctx, "Key is required")
	}
	if err := store.Database.DeleteAgentMemoryFact(ctx.Context(), scope, key); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(ctx, "Fact not found")
		}
		return toastError(ctx, "Failed to delete fact")
	}
	chatagent.InvalidatePromptCache()
	return renderAgentMemoryTable(ctx, scope)
}

func agentMemoryDeleteFact(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := strings.TrimSpace(ctx.Query("scope"))
	key := strings.TrimSpace(ctx.Query("key"))
	if scope == "" || key == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("scope and key are required")))
	}
	if err := store.Database.DeleteAgentMemoryFact(ctx.Context(), scope, key); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(protocol.NewFailedResponse(errors.New("fact not found")))
		}
		return types.WrapError(types.ErrInternal, "delete memory fact", err)
	}
	chatagent.InvalidatePromptCache()
	return ctx.JSON(protocol.NewSuccessResponse(map[string]string{"status": "deleted"}))
}

func updateExistingAgentMemoryFact(ctx context.Context, scope, key, value string, pinned bool) (*gen.AgentMemoryFact, error) {
	if _, err := store.Database.GetAgentMemoryFact(ctx, scope, key); err != nil {
		return nil, err
	}
	return store.Database.UpsertAgentMemoryFact(ctx, store.AgentMemoryFactUpsert{
		Scope:  scope,
		Key:    key,
		Value:  value,
		Pinned: pinned,
	})
}

func renderAgentMemoryTable(ctx fiber.Ctx, scope string) error {
	items, err := listAgentMemoryFactModels(ctx.Context(), scope)
	if err != nil {
		return toastError(ctx, "Failed to refresh memory facts")
	}
	ctx.Type("html")
	return partials.AgentMemoryTable(items, scope).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentMemoryScopeQuery(ctx fiber.Ctx) string {
	scope := strings.TrimSpace(ctx.Query("scope"))
	if scope == "" {
		return "default"
	}
	return scope
}

func listAgentMemoryFactModels(ctx context.Context, scope string) ([]model.AgentMemoryFact, error) {
	rows, err := store.Database.ListAgentMemoryFacts(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := make([]model.AgentMemoryFact, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentMemoryFactModelFromGen(row))
	}
	return out, nil
}

func agentMemoryFactModelFromGen(row *gen.AgentMemoryFact) model.AgentMemoryFact {
	if row == nil {
		return model.AgentMemoryFact{}
	}
	return model.AgentMemoryFact{
		ID:        row.ID,
		Scope:     row.Scope,
		Key:       row.Key,
		Value:     row.Value,
		Pinned:    row.Pinned,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
