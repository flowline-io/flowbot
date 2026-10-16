package lineage

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var auditRules = []webservice.Rule{
	webservice.Get("", listAudit),
}

func listAudit(ctx fiber.Ctx) error {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return types.Errorf(types.ErrUnauthorized, "missing caller")
	}
	opts, err := auditOptions(ctx, string(rc.UID), auth.HasScope(rc.Scopes, auth.ScopeAdmin))
	if err != nil {
		return err
	}
	rows, cursor, err := store.AuditStoreFromDB().List(ctx.Context(), opts)
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

func auditOptions(ctx fiber.Ctx, uid string, admin bool) (store.ListAuditOptions, error) {
	subject, err := VisibleSubject(uid, admin, ctx.Query("subject_id"))
	if err != nil {
		return store.ListAuditOptions{}, err
	}
	limit := 0
	if raw := ctx.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return store.ListAuditOptions{}, &types.Error{Kind: types.ErrInvalidArgument, Message: "limit must be a positive integer", Field: "limit"}
		}
		limit = n
	}
	since, err := queryTime(ctx, "since")
	if err != nil {
		return store.ListAuditOptions{}, err
	}
	until, err := queryTime(ctx, "until")
	if err != nil {
		return store.ListAuditOptions{}, err
	}
	return store.ListAuditOptions{
		Action:     ctx.Query("action"),
		SubjectID:  subject,
		TargetType: ctx.Query("target_type"),
		Result:     ctx.Query("result"),
		Since:      since,
		Until:      until,
		Limit:      limit,
		Cursor:     ctx.Query("cursor"),
	}, nil
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

// VisibleSubject returns the uid filter for an audit list.
// A non-admin caller is limited to their own uid. admin with an empty subject lists every uid.
func VisibleSubject(uid string, admin bool, subjectID string) (string, error) {
	if !admin {
		if subjectID != "" && subjectID != uid {
			return "", types.Errorf(types.ErrForbidden, "subject_id is outside the caller")
		}
		return uid, nil
	}
	return subjectID, nil
}

// ListForCaller returns audit rows visible to uid. admin skips the subject restriction when subjectID is empty.
func ListForCaller(ctx context.Context, uid string, admin bool, subjectID string, opts store.ListAuditOptions) ([]store.AuditRecord, string, error) {
	subject, err := VisibleSubject(uid, admin, subjectID)
	if err != nil {
		return nil, "", err
	}
	opts.SubjectID = subject
	return store.AuditStoreFromDB().List(ctx, opts)
}
