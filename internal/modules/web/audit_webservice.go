package web

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

// auditWebserviceRules is mounted under /service/audit (group name "audit"), not /service/web.
var auditWebserviceRules = []webservice.Rule{
	webservice.Get("", listAudit),
}

func mountAuditRoutes(app *fiber.App) {
	module.Webservice(app, "audit", auditWebserviceRules)
}

// MountAuditForE2E mounts GET /service/audit onto app.
func MountAuditForE2E(app *fiber.App) {
	mountAuditRoutes(app)
}

func listAudit(ctx fiber.Ctx) error {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return types.Errorf(types.ErrUnauthorized, "missing caller")
	}
	opts, err := auditListOptions(ctx)
	if err != nil {
		return err
	}
	rows, cursor, err := store.AuditStoreFromDB().ListForCaller(ctx.Context(), store.AuditCaller{
		UID:       string(rc.UID),
		Admin:     auth.HasScope(rc.Scopes, auth.ScopeAdmin),
		SubjectID: ctx.Query("subject_id"),
	}, opts)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.Item())
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{
		"items": items,
		"page":  map[string]any{"cursor": cursor},
	}))
}

func auditListOptions(ctx fiber.Ctx) (store.ListAuditOptions, error) {
	limit := 0
	if raw := ctx.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return store.ListAuditOptions{}, &types.Error{Kind: types.ErrInvalidArgument, Message: "limit must be a positive integer", Field: "limit"}
		}
		limit = n
	}
	since, err := auditQueryTime(ctx, "since")
	if err != nil {
		return store.ListAuditOptions{}, err
	}
	until, err := auditQueryTime(ctx, "until")
	if err != nil {
		return store.ListAuditOptions{}, err
	}
	return store.ListAuditOptions{
		Action:     ctx.Query("action"),
		TargetType: ctx.Query("target_type"),
		Result:     ctx.Query("result"),
		Since:      since,
		Until:      until,
		Limit:      limit,
		Cursor:     ctx.Query("cursor"),
	}, nil
}

func auditQueryTime(ctx fiber.Ctx, name string) (time.Time, error) {
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
