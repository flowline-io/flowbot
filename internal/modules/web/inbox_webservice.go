package web

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var inboxWebserviceRules = []webservice.Rule{
	webservice.Get("/inbox", inboxPage),
	webservice.Get("/inbox/list", inboxList),
	webservice.Get("/inbox-badge", inboxBadge),
	webservice.Post("/inbox/:id/read", inboxMarkRead),
	webservice.Get("/inbox/:id/open", inboxOpen),
}

func inboxPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	filter := strings.TrimSpace(ctx.Query("filter"))
	if filter == "" {
		filter = "unread"
	}
	ctx.Type("html")
	return pages.InboxPage(filter).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func inboxList(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	notifypkg.TouchPresence(uid)
	return renderInboxList(ctx, uid)
}

func inboxBadge(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	count := 0
	if uid != "" {
		notifypkg.TouchPresence(uid)
		if ns := notifypkg.GetNotifyStore(); ns != nil {
			if n, err := ns.CountUnread(ctx.Context(), uid, notifypkg.ChannelInapp, "success"); err == nil {
				count = n
			}
		}
	}
	ctx.Type("html")
	return partials.InboxCountBadge(count).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func inboxMarkRead(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.SendString("Invalid ID")
	}
	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	rec, err := ns.GetRecord(ctx.Context(), id)
	if err != nil || rec == nil {
		ctx.Type("html")
		return partials.EmptyState("Record not found").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if rec.UID != uid {
		ctx.Status(fiber.StatusForbidden)
		return ctx.SendString("Not your notification")
	}
	if err := ns.MarkRead(ctx.Context(), uid, id); err != nil {
		setShowToast(ctx, "error", "Failed to mark as read")
		return renderInboxList(ctx, uid)
	}
	return renderInboxList(ctx, uid)
}

func inboxOpen(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		return redirectToLogin(ctx)
	}
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.SendString("Invalid ID")
	}
	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Status(fiber.StatusServiceUnavailable)
		return ctx.SendString("Store not available")
	}
	rec, err := ns.GetRecord(ctx.Context(), id)
	if err != nil || rec == nil || rec.UID != uid {
		ctx.Status(fiber.StatusNotFound)
		return ctx.SendString("Not found")
	}
	if err := ns.MarkRead(ctx.Context(), uid, id); err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.SendString("Failed to mark as read")
	}
	target := "/service/web/inbox"
	if rec.PayloadSnapshot != nil {
		if u, ok := rec.PayloadSnapshot["url"].(string); ok {
			if safe, ok := safeInboxRedirectURL(strings.TrimSpace(u)); ok {
				target = safe
			}
		}
	}
	return ctx.Redirect().To(target)
}

func renderInboxList(ctx fiber.Ctx, uid string) error {
	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	filter := strings.TrimSpace(ctx.Query("filter"))
	if filter == "" {
		filter = "unread"
	}
	opts := store.ListNotifyRecordsOptions{
		Limit:   20,
		Cursor:  ctx.Query("cursor"),
		Channel: notifypkg.ChannelInapp,
		Status:  "success",
	}
	if filter != "all" {
		opts.UnreadOnly = true
	}
	records, next, err := ns.ListRecords(ctx.Context(), uid, opts)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load inbox").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.InboxList(partials.InboxListParams{
		Filter: filter,
		Items:  inboxItemsFromRecords(records),
		Cursor: next,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func inboxItemsFromRecords(records []*gen.NotificationRecord) []partials.InboxItem {
	out := make([]partials.InboxItem, 0, len(records))
	for _, rec := range records {
		if rec == nil {
			continue
		}
		item := partials.InboxItem{
			ID:        rec.ID,
			Summary:   rec.Summary,
			CreatedAt: rec.CreatedAt,
			Unread:    rec.ReadAt == nil,
		}
		if rec.PayloadSnapshot != nil {
			if t, ok := rec.PayloadSnapshot["title"].(string); ok {
				item.Title = t
			}
			if u, ok := rec.PayloadSnapshot["url"].(string); ok {
				item.URL = u
			}
		}
		out = append(out, item)
	}
	return out
}

// safeInboxRedirectURL allows only relative /service/web paths (no scheme/host open redirect).
func safeInboxRedirectURL(raw string) (string, bool) {
	if raw == "" || strings.HasPrefix(raw, "//") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" || u.Host != "" {
		return "", false
	}
	if !strings.HasPrefix(u.Path, "/service/web/") {
		return "", false
	}
	return u.RequestURI(), true
}
