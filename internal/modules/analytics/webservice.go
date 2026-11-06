package analytics

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

// LineageCharge charges the shared MCP lineage rate-limit buckets for API-token
// REST calls. Wired by the MCP package at process start; nil skips the charge.
type LineageCharge func(ctx context.Context, tokenID int64, kind string) error

var lineageCharge LineageCharge

// SetLineageCharge installs the shared lineage REST rate limiter.
func SetLineageCharge(fn LineageCharge) {
	lineageCharge = fn
}

var lineageWebserviceRules = []webservice.Rule{
	webservice.Get("/trace", lineageTrace),
	webservice.Get("/search", lineageSearch),
}

var flowsWebserviceRules = []webservice.Rule{
	webservice.Get("/graph", flowGraph),
}

var eventsWebserviceRules = []webservice.Rule{
	webservice.Get("/timeline", eventsTimeline),
}

func (moduleHandler) Webservice(app *fiber.App) {
	if !handler.initialized {
		return
	}
	module.Webservice(app, "lineage", lineageWebserviceRules)
	module.Webservice(app, "flows", flowsWebserviceRules)
	module.Webservice(app, "events", eventsWebserviceRules)
}

func requireCaller(ctx fiber.Ctx) (*route.RequestContext, error) {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return nil, types.Errorf(types.ErrUnauthorized, "missing caller")
	}
	if lineageCharge != nil {
		if err := lineageCharge(ctx.Context(), rc.TokenID, rc.Kind); err != nil {
			return nil, err
		}
	}
	return rc, nil
}

func requireService(ctx fiber.Ctx) (*Service, *route.RequestContext, error) {
	rc, err := requireCaller(ctx)
	if err != nil {
		return nil, nil, err
	}
	svc := ActiveService()
	if svc == nil {
		return nil, nil, types.Errorf(types.ErrUnavailable, "analytics service not ready")
	}
	return svc, rc, nil
}

func lineageTrace(ctx fiber.Ctx) error {
	svc, _, err := requireService(ctx)
	if err != nil {
		return err
	}
	q := types.LineageTraceQuery{
		App:           ctx.Query("app"),
		EntityID:      ctx.Query("entity_id"),
		EventID:       ctx.Query("event_id"),
		CorrelationID: ctx.Query("correlation_id"),
		Direction:     types.NormalizeLineageTraceDirection(types.LineageTraceDirection(ctx.Query("direction"))),
	}
	if raw := ctx.Query("depth"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return &types.Error{Kind: types.ErrInvalidArgument, Message: "depth must be a positive integer", Field: "depth"}
		}
		q.Depth = n
	}
	out, err := svc.Trace(ctx.Context(), q)
	if err != nil {
		if strings.Contains(err.Error(), "exactly one of") {
			return &types.Error{Kind: types.ErrInvalidArgument, Message: err.Error()}
		}
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(out))
}

func lineageSearch(ctx fiber.Ctx) error {
	svc, _, err := requireService(ctx)
	if err != nil {
		return err
	}
	q := types.LineageSearchQuery{
		Query:  ctx.Query("query"),
		Cursor: ctx.Query("cursor"),
	}
	if raw := ctx.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return &types.Error{Kind: types.ErrInvalidArgument, Message: "limit must be a positive integer", Field: "limit"}
		}
		q.Limit = n
	}
	out, err := svc.Search(ctx.Context(), q)
	if err != nil {
		return err
	}
	page := map[string]any{}
	if out.NextCursor != "" {
		page["next_cursor"] = out.NextCursor
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{
		"items": out.Items,
		"page":  page,
	}))
}

func flowGraph(ctx fiber.Ctx) error {
	svc, _, err := requireService(ctx)
	if err != nil {
		return err
	}
	q := types.FlowGraphQuery{
		App:           ctx.Query("app"),
		IncludeStatic: queryBoolDefault(ctx, "include_static", true),
	}
	since, err := queryTime(ctx, "since")
	if err != nil {
		return err
	}
	until, err := queryTime(ctx, "until")
	if err != nil {
		return err
	}
	q.Since = since
	q.Until = until
	out, err := svc.FlowGraph(ctx.Context(), q)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(out))
}

func eventsTimeline(ctx fiber.Ctx) error {
	svc, rc, err := requireService(ctx)
	if err != nil {
		return err
	}
	q := types.EventsTimelineQuery{
		App:            ctx.Query("app"),
		EventType:      ctx.Query("event_type"),
		CorrelationID:  ctx.Query("correlation_id"),
		Cursor:         ctx.Query("cursor"),
		IncludePayload: queryBool(ctx, "include_payload"),
		Scopes:         rc.Scopes,
	}
	if !auth.HasScope(rc.Scopes, auth.ScopeAdmin) {
		q.CallerUID = string(rc.UID)
	}
	if raw := ctx.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return &types.Error{Kind: types.ErrInvalidArgument, Message: "limit must be a positive integer", Field: "limit"}
		}
		q.Limit = n
	}
	since, err := queryTime(ctx, "since")
	if err != nil {
		return err
	}
	until, err := queryTime(ctx, "until")
	if err != nil {
		return err
	}
	q.Since = since
	q.Until = until
	out, err := svc.Timeline(ctx.Context(), q)
	if err != nil {
		return err
	}
	page := map[string]any{}
	if out.NextCursor != "" {
		page["next_cursor"] = out.NextCursor
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{
		"items": out.Items,
		"page":  page,
	}))
}

func queryTime(ctx fiber.Ctx, name string) (time.Time, error) {
	raw := ctx.Query(name)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, &types.Error{Kind: types.ErrInvalidArgument, Message: name + " must be RFC3339", Field: name}
	}
	return t, nil
}

func queryBool(ctx fiber.Ctx, name string) bool {
	raw := strings.TrimSpace(ctx.Query(name))
	return strings.EqualFold(raw, "true") || raw == "1"
}

func queryBoolDefault(ctx fiber.Ctx, name string, def bool) bool {
	if !ctx.Request().URI().QueryArgs().Has(name) {
		return def
	}
	return queryBool(ctx, name)
}
