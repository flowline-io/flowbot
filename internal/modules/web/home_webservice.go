package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var homeWebserviceRules = []webservice.Rule{
	webservice.Get("/home", homePage, route.WithNotAuth()),
	webservice.Get("/home/dashboard", homeDashboardPartial, route.WithNotAuth()),
	webservice.Get("/home/token-usage", homeTokenUsage, route.WithNotAuth()),
	webservice.Get("/session-badge", sessionBadge, route.WithNotAuth()),
}

func homePage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	// Dashboard is cheap (short pings + store counts); SSR avoids HTMX skeleton flash.
	return pages.HomePage(buildHomeDashboard(ctx.Context())).Render(context.Background(), ctx.Response().BodyWriter())
}

func homeDashboardPartial(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	d := buildHomeDashboard(ctx.Context())
	return partials.HomeDashboardBlock(d).Render(context.Background(), ctx.Response().BodyWriter())
}

// buildHomeDashboard assembles summary stats and an optional setup checklist for Home.
// Intentionally avoids gatherHealthzData (per-capability probes) and runtime Status fan-out.
func buildHomeDashboard(ctx context.Context) partials.HomeDashboard {
	d := partials.HomeDashboard{}

	pingCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	if store.Database != nil && store.Database.IsOpen() {
		_, err := store.Database.Ping(pingCtx)
		d.PostgresOK = err == nil
	}
	if rs := cache.DefaultRedisStore(); rs != nil {
		_, err := rs.Ping(pingCtx)
		d.RedisOK = err == nil
	}

	if store.Database != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok && client != nil {
			ps := store.NewPipelineStore(client)
			since7d := time.Now().Add(-7 * 24 * time.Hour)
			if stats, err := ps.PipelineStats(ctx, "", since7d, "day"); err == nil && stats != nil {
				d.PipelineTotal = stats.Summary.TotalPipelines
				d.PipelineOK = stats.Summary.SuccessfulRuns
				d.PipelineFailed = stats.Summary.FailedRuns
			}
			es := store.NewEventStore(client)
			since24h := time.Now().Add(-24 * time.Hour)
			if n, err := es.CountDataEvents(ctx, store.ListDataEventsOptions{TimeStart: &since24h}); err == nil {
				d.Events24h = n
			}
		}
	}

	apps := homelab.DefaultRegistry.List()
	d.HubAppsTotal = len(apps)
	for _, a := range apps {
		if a.Status == homelab.AppStatusRunning {
			d.HubAppsRunning++
		}
	}

	d.Checklist = buildHomeChecklist(ctx, d)
	return d
}

func buildHomeChecklist(ctx context.Context, d partials.HomeDashboard) []partials.HomeChecklistItem {
	hasPipelines := d.PipelineTotal > 0
	hasHub := d.HubAppsTotal > 0
	hasAgentsReady := false
	if store.Database != nil {
		if skills, err := store.Database.ListAgentSkills(ctx, false); err == nil && len(skills) > 0 {
			hasAgentsReady = true
		}
	}
	items := []partials.HomeChecklistItem{
		{
			Done:   hasHub,
			Title:  "Connect Hub apps",
			Detail: "Start or register integrations this instance will orchestrate.",
			Href:   "/service/web/hub",
			CTA:    "Open Hub",
			TestID: "home-check-hub",
		},
		{
			Done:   hasPipelines,
			Title:  "Create a pipeline",
			Detail: "Automate reactions to data events.",
			Href:   "/service/web/pipelines",
			CTA:    "Open Pipelines",
			TestID: "home-check-pipelines",
		},
		{
			Done:   hasAgentsReady,
			Title:  "Try Agents",
			Detail: "Chat with an agent and configure skills when ready.",
			Href:   "/service/web/agents",
			CTA:    "Open Agents",
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
	return partials.TokenUsage(stats).Render(context.Background(), c.Response().BodyWriter())
}

func invalidTokenUsageRequest(c fiber.Ctx, err error) error {
	if errors.Is(err, types.ErrInvalidArgument) {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	return c.Status(http.StatusBadRequest).SendString(err.Error())
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
	return partials.SessionBadge(username, expires).Render(context.Background(), ctx.Response().BodyWriter())
}
