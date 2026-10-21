package web

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var analyticsWebserviceRules = []webservice.Rule{
	webservice.Get("/analytics", analyticsOverviewPage),
	webservice.Get("/analytics/token-usage", analyticsTokenUsage),
	webservice.Get("/analytics/events", analyticsEventsPage),
	webservice.Get("/analytics/events/panel", analyticsEventsPanel),
	webservice.Get("/analytics/activity", analyticsActivityPage),
	webservice.Get("/analytics/activity/panel", analyticsActivityPanel),
	webservice.Get("/analytics/ops/runs", analyticsOpsRunsPage),
	webservice.Get("/analytics/ops/runs/panel", analyticsOpsRunsPanel),
}

func analyticsOverviewPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	uid := getUID(c)
	if uid == "" {
		return redirectToLogin(c)
	}
	svc := analyticsService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "analytics unavailable")
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	overview, err := svc.Overview(c.Context(), uid, ar.Since, ar.Until, ar.ActiveRange, ar.RangeLabel)
	if err != nil {
		return types.Errorf(types.ErrInternal, "analytics overview: %v", err)
	}
	c.Type("html")
	return pages.AnalyticsOverviewPage(c.Context(), overview, ar.Query).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsTokenUsage(c fiber.Ctx) error {
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
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidTokenUsageRequest(c, err)
	}
	svc := analyticsService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "analytics unavailable")
	}
	stats, err := svc.TokenUsage(c.Context(), uid, ar.Since, ar.Until, groupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "token usage stats: %v", err)
	}
	stats.RangeLabel = ar.RangeLabel
	stats.ActiveRange = ar.ActiveRange
	stats.GroupBy = groupBy

	accept := c.Get("Accept", "")
	if strings.Contains(accept, "application/json") {
		return c.JSON(stats)
	}
	c.Type("html")
	if c.Get("HX-Request") != "" {
		return partials.TokenUsage(stats).Render(c.Context(), c.Response().BodyWriter())
	}
	return pages.AnalyticsTokenPage(c.Context(), stats, ar.Query).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsEventsPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	c.Type("html")
	return pages.AnalyticsEventsPage(c.Context(), ar.Query).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsEventsPanel(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	uid := getUID(c)
	if uid == "" {
		return redirectToLogin(c)
	}
	groupBy, err := types.NormalizeEventStatsGroupBy(c.Query("groupBy", ""))
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	svc := analyticsService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "analytics unavailable")
	}
	stats, err := svc.EventStats(c.Context(), uid, ar.Since, ar.Until, groupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "event stats: %v", err)
	}
	stats.RangeLabel = ar.RangeLabel
	stats.ActiveRange = ar.ActiveRange
	stats.GroupBy = groupBy
	c.Type("html")
	return partials.AnalyticsEvents(stats).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsActivityPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	c.Type("html")
	return pages.AnalyticsActivityPage(c.Context(), ar.Query).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsActivityPanel(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	uid := getUID(c)
	if uid == "" {
		return redirectToLogin(c)
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	svc := analyticsService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "analytics unavailable")
	}
	stats, err := svc.ActivityStats(c.Context(), uid, ar.Since, ar.Until)
	if err != nil {
		return types.Errorf(types.ErrInternal, "activity stats: %v", err)
	}
	stats.RangeLabel = ar.RangeLabel
	stats.ActiveRange = ar.ActiveRange
	c.Type("html")
	return partials.AnalyticsActivity(stats).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsOpsRunsPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	c.Type("html")
	return pages.AnalyticsOpsRunsPage(c.Context(), ar.Query).Render(c.Context(), c.Response().BodyWriter())
}

func analyticsOpsRunsPanel(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	ar, err := resolveAnalyticsRange(c)
	if err != nil {
		return invalidAnalyticsRange(c)
	}
	groupBy := c.Query("groupBy", "day")
	if groupBy != "day" && groupBy != "week" && groupBy != "month" {
		groupBy = "day"
	}
	svc := analyticsService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "analytics unavailable")
	}
	pipe, wf, fn, err := svc.OpsRunsHealth(c.Context(), ar.Since, ar.Until, groupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "ops runs health: %v", err)
	}
	if pipe == nil {
		pipe = &types.PipelineStats{}
	}
	if wf == nil {
		wf = &types.WorkflowStats{}
	}
	if fn == nil {
		fn = &types.FunctionStats{}
	}
	c.Type("html")
	return partials.AnalyticsOpsRuns(pipe, wf, fn, ar.ActiveRange, ar.RangeLabel, groupBy).Render(c.Context(), c.Response().BodyWriter())
}
