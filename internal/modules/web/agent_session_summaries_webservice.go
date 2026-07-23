package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var agentSessionSummariesWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-session-summaries", agentSessionSummariesPage, route.WithNotAuth()),
	webservice.Get("/agent-session-summaries/list", agentSessionSummariesTable, route.WithNotAuth()),
	webservice.Post("/agent-session-summaries/:session/retry", agentSessionSummaryRetry, route.WithNotAuth()),
}

func agentSessionSummariesPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	q := strings.TrimSpace(ctx.Query("q"))
	items, err := listAgentSessionSummaryModels(ctx.Context(), q)
	if err != nil {
		return types.WrapError(types.ErrInternal, "list session summaries", err)
	}
	ctx.Type("html")
	return pages.AgentSessionSummariesPage(items, q).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSessionSummariesTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	q := strings.TrimSpace(ctx.Query("q"))
	items, err := listAgentSessionSummaryModels(ctx.Context(), q)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load session summaries")
	}
	ctx.Type("html")
	return partials.AgentSessionSummariesTable(items, q).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSessionSummaryRetry(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(ctx.Params("session"))
	if sessionID == "" {
		return toastError(ctx, "Session id is required")
	}
	if err := chatagent.RetrySessionSummary(ctx.Context(), sessionID); err != nil {
		return toastError(ctx, "Failed to retry summary")
	}
	q := strings.TrimSpace(ctx.Query("q"))
	items, err := listAgentSessionSummaryModels(ctx.Context(), q)
	if err != nil {
		return toastError(ctx, "Failed to refresh summaries")
	}
	ctx.Type("html")
	return partials.AgentSessionSummariesTable(items, q).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func listAgentSessionSummaryModels(ctx context.Context, q string) ([]model.AgentSessionSummary, error) {
	rows, err := store.Database.ListAgentSessionSummaries(ctx, store.AgentSessionSummaryListFilter{
		Scope: "default",
		Q:     q,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.AgentSessionSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentSessionSummaryModelFromGen(row))
	}
	return out, nil
}

func agentSessionSummaryModelFromGen(row *gen.AgentSessionSummary) model.AgentSessionSummary {
	if row == nil {
		return model.AgentSessionSummary{}
	}
	item := model.AgentSessionSummary{
		ID:          row.ID,
		SessionFlag: row.SessionFlag,
		Scope:       row.Scope,
		Title:       row.Title,
		Summary:     row.Summary,
		Status:      row.Status,
		Error:       row.Error,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		ClaimedAt:   row.ClaimedAt,
	}
	if item.Status == "" {
		item.Status = schema.AgentSessionSummaryPending
	}
	return item
}
