package web

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notificationWebserviceRules = []webservice.Rule{
	webservice.Get("/notifications", notificationsPage, route.WithNotAuth()),
	webservice.Get("/notifications/list", notificationsTable, route.WithNotAuth()),
	webservice.Post("/notifications/:id/retry", retryNotification, route.WithNotAuth()),
}

func getUID(ctx fiber.Ctx) string {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return ""
	}
	return rc.UID.String()
}

func notificationsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	records, nextCursor, err := ns.ListRecords(ctx.Context(), uid, store.ListNotifyRecordsOptions{Limit: 20})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load notifications").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return pages.NotificationsPage(pages.NotificationsPageParams{
		Records:    records,
		NextCursor: nextCursor,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notificationsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	cursor := ctx.Query("cursor")

	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	records, nextCursor, err := ns.ListRecords(ctx.Context(), uid, store.ListNotifyRecordsOptions{
		Limit:  20,
		Cursor: cursor,
	})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load notifications").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.NotificationsTable(records, nextCursor).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func retryNotification(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	idStr := ctx.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
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
	if string(rec.Status) != "failed" {
		ctx.Type("html")
		return partials.EmptyState("Only failed notifications can be retried").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	payload := make(map[string]any)
	if rec.PayloadSnapshot != nil {
		for k, v := range rec.PayloadSnapshot {
			payload[k] = v
		}
	}

	notifyUid := types.Uid(rec.UID)
	if err := notifypkg.GatewaySend(context.Background(), notifyUid, rec.TemplateID, []string{rec.Channel}, payload); err != nil {
		ctx.Type("html")
		return partials.EmptyState("Retry failed: "+err.Error()).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	// Wait briefly for the async record goroutine to persist the retry outcome
	time.Sleep(50 * time.Millisecond)

	records, nextCursor, listErr := ns.ListRecords(context.Background(), uid, store.ListNotifyRecordsOptions{Limit: 20})
	if listErr != nil {
		ctx.Type("html")
		return partials.EmptyState("Retried but failed to reload").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.NotificationsTable(records, nextCursor).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}
