package web

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var homeWebserviceRules = []webservice.Rule{
	webservice.Get("/home", homePage),
	webservice.Get("/home/dashboard", homeDashboardPartial),
	webservice.Get("/home/token-usage", homeTokenUsage),
	webservice.Get("/session-badge", sessionBadge),
	webservice.Get("/approval-badge", approvalBadge),
}

func homePage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	// Dashboard is cheap (short pings + store counts); SSR avoids HTMX skeleton flash.
	return pages.HomePage(ctx.Context(), buildHomeDashboard(ctx.Context())).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func homeDashboardPartial(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	d := buildHomeDashboard(ctx.Context())
	return partials.HomeDashboardBlock(ctx.Context(), d).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// buildHomeDashboard assembles summary stats and an optional setup checklist for Home.
// Intentionally avoids gatherHealthzData (per-capability probes) and runtime Status fan-out.
func buildHomeDashboard(ctx context.Context) partials.HomeDashboard {
	d := partials.HomeDashboard{}

	if store.Database != nil {
		active := int(schema.ChatSessionActive)
		if n, err := store.ChatStoreFromDB().CountChatSessions(ctx, store.ListChatSessionsOptions{State: &active}); err == nil {
			d.ActiveSessions = n
		}
		if store.Database.GetClient() != nil {
			since7d := time.Now().Add(-7 * 24 * time.Hour)
			ps := store.PipelineStoreFromDB()
			if stats, err := ps.PipelineStats(ctx, "", since7d, "day"); err == nil && stats != nil {
				d.PipelineTotal = stats.Summary.TotalPipelines
				d.PipelineOK = stats.Summary.SuccessfulRuns
				d.PipelineFailed = stats.Summary.FailedRuns
			}
			ws := store.WorkflowStoreFromDB()
			if stats, err := ws.WorkflowStats(ctx, "", since7d, "day"); err == nil && stats != nil {
				d.WorkflowTotal = stats.Summary.TotalWorkflows
				d.WorkflowOK = stats.Summary.SuccessfulRuns
				d.WorkflowFailed = stats.Summary.FailedRuns
			}
			es := store.EventStoreFromDB()
			since24h := time.Now().Add(-24 * time.Hour)
			if n, err := es.CountDataEvents(ctx, store.ListDataEventsOptions{TimeStart: &since24h}); err == nil {
				d.Events24h = n
			}
		}
	}

	d.Checklist = buildHomeChecklist(ctx, d)
	return d
}

func buildHomeChecklist(ctx context.Context, d partials.HomeDashboard) []partials.HomeChecklistItem {
	hasPipelines := d.PipelineTotal > 0
	hasWorkflows := d.WorkflowTotal > 0
	hasAgentsReady := d.ActiveSessions > 0
	if !hasAgentsReady && store.Database != nil {
		if skills, err := store.AgentStoreFromDB().ListAgentSkills(ctx, false); err == nil && len(skills) > 0 {
			hasAgentsReady = true
		}
	}
	items := []partials.HomeChecklistItem{
		{
			Done:   hasWorkflows,
			Title:  i18n.T(ctx, "home.checklist.workflow.title"),
			Detail: i18n.T(ctx, "home.checklist.workflow.detail"),
			Href:   "/service/web/workflows",
			CTA:    i18n.T(ctx, "home.checklist.workflow.cta"),
			TestID: "home-check-workflows",
		},
		{
			Done:   hasPipelines,
			Title:  i18n.T(ctx, "home.checklist.pipeline.title"),
			Detail: i18n.T(ctx, "home.checklist.pipeline.detail"),
			Href:   "/service/web/pipelines",
			CTA:    i18n.T(ctx, "common.open_pipelines"),
			TestID: "home-check-pipelines",
		},
		{
			Done:   hasAgentsReady,
			Title:  i18n.T(ctx, "home.checklist.agents.title"),
			Detail: i18n.T(ctx, "home.checklist.agents.detail"),
			Href:   "/service/web/agents",
			CTA:    i18n.T(ctx, "home.checklist.agents.cta"),
			TestID: "home-check-agents",
		},
	}
	allDone := true
	for _, it := range items {
		if !it.Done {
			allDone = false
			break
		}
	}
	if allDone {
		return nil
	}
	return items
}

func homeTokenUsage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	uid := getUID(c)
	if uid == "" {
		return redirectToLogin(c)
	}

	groupBy, err := types.NormalizeTokenUsageGroupBy(c.Query("groupBy", ""))
	if err != nil {
		return invalidTokenUsageRequest(c, err)
	}

	since, until, activeRange, rangeLabel, err := types.ResolveTokenUsageRange(
		c.Query("range", ""),
		c.Query("since", ""),
		c.Query("until", ""),
		time.Now().UTC(),
	)
	if err != nil {
		return invalidTokenUsageRequest(c, err)
	}

	usageStore := store.NewLLMUsageStoreFromDatabase()
	if usageStore == nil {
		return types.Errorf(types.ErrInternal, "store not available")
	}

	stats, err := usageStore.TokenUsageStats(c.Context(), uid, since, until, groupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "token usage stats: %v", err)
	}
	stats.RangeLabel = rangeLabel
	stats.ActiveRange = activeRange
	stats.GroupBy = groupBy

	accept := c.Get("Accept", "")
	if strings.Contains(accept, "application/json") {
		return c.JSON(stats)
	}
	c.Type("html")
	return partials.TokenUsage(stats).Render(c.Context(), c.Response().BodyWriter())
}

func invalidTokenUsageRequest(c fiber.Ctx, _ error) error {
	return c.Status(http.StatusBadRequest).SendString(webMsg(c, "error.token_usage.invalid_request"))
}

// sessionBadge renders a compact navbar identity fragment for the current web session.
func sessionBadge(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rc := route.GetRequestContext(ctx)
	username := "operator"
	if rc != nil {
		if uid := strings.TrimPrefix(rc.UID.String(), "user-"); uid != "" {
			username = uid
		}
	}
	expires := ""
	token := ctx.Cookies("accessToken")
	if token != "" {
		if p, err := route.LookupAccessToken(context.Background(), token); err == nil && p.ID > 0 && !p.ExpiredAt.IsZero() {
			remaining := time.Until(p.ExpiredAt).Round(time.Minute)
			if remaining > 0 {
				expires = remaining.String() + " left"
			} else {
				expires = "expired"
			}
		}
	}
	ctx.Type("html")
	return partials.SessionBadge(username, expires).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// approvalBadge renders a compact navbar/home count for sessions awaiting tool approval.
func approvalBadge(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	count, err := pendingApprovalSessionCount()
	if err != nil {
		return err
	}
	return partials.ApprovalCountBadge(count).Render(ctx.Context(), ctx.Response().BodyWriter())
}
