package web

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/i18n"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notificationWebserviceRules = []webservice.Rule{
	webservice.Get("/notifications", notifySettingsPage),
	webservice.Get("/notifications/list", notificationsTable),
	webservice.Post("/notifications/:id/retry", retryNotification),
	webservice.Post("/notifications/:id/read", markNotificationRead),
}

func notificationsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.not_authenticated")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.store_unavailable")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	return renderNotificationsTable(ctx, ns, uid)
}

func retryNotification(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.not_authenticated")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	idStr := ctx.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.SendString(webMsg(ctx, "error.notification.invalid_id"))
	}

	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.store_unavailable")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	rec, err := ns.GetRecord(ctx.Context(), id)
	if err != nil || rec == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.record_not_found")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if rec.UID != uid {
		ctx.Status(fiber.StatusForbidden)
		return ctx.SendString(webMsg(ctx, "error.notification.not_yours"))
	}
	if rec.Status != "failed" {
		setShowToastKey(ctx, "error", "toast.notification.retry_only_failed")
		return renderNotificationsTable(ctx, ns, uid)
	}

	if notifypkg.IsConnectivityTestTemplate(rec.TemplateID) {
		if err := retryConnectivityTest(ctx.Context(), ns, uid, rec.Channel); err != nil {
			setShowToast(ctx, "error", webMsgData(ctx, "toast.notification.retry_failed", map[string]any{"Error": err.Error()}))
			return renderNotificationsTable(ctx, ns, uid)
		}
		setShowToastKey(ctx, "success", "toast.notification.connectivity_retest_succeeded")
		return renderNotificationsTable(ctx, ns, uid)
	}

	payload := make(map[string]any)
	if rec.PayloadSnapshot != nil {
		maps.Copy(payload, rec.PayloadSnapshot)
	}

	notifyUid := types.Uid(rec.UID)
	if err := notifypkg.GatewaySend(context.Background(), notifyUid, rec.TemplateID, []string{rec.Channel}, payload); err != nil {
		setShowToast(ctx, "error", webMsgData(ctx, "toast.notification.retry_failed", map[string]any{"Error": err.Error()}))
		return renderNotificationsTable(ctx, ns, uid)
	}

	// Wait briefly for the async record goroutine to persist the retry outcome
	time.Sleep(50 * time.Millisecond)
	setShowToastKey(ctx, "success", "toast.notification.notification_retried")
	return renderNotificationsTable(ctx, ns, uid)
}

func markNotificationRead(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.not_authenticated")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.SendString(webMsg(ctx, "error.notification.invalid_id"))
	}

	ns := notifypkg.GetNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.store_unavailable")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	rec, err := ns.GetRecord(ctx.Context(), id)
	if err != nil || rec == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.record_not_found")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if rec.UID != uid {
		ctx.Status(fiber.StatusForbidden)
		return ctx.SendString(webMsg(ctx, "error.notification.not_yours"))
	}
	if err := ns.MarkRead(ctx.Context(), uid, id); err != nil {
		setShowToastKey(ctx, "error", "toast.notification.mark_read_failed")
		return renderNotificationsTable(ctx, ns, uid)
	}
	return renderNotificationsTable(ctx, ns, uid)
}

// renderNotificationsTable reloads and renders the notifications table fragment.
func renderNotificationsTable(ctx fiber.Ctx, ns notifypkg.NotifyRecords, uid string) error {
	opts := notifyHistoryListOptions(ctx)
	records, nextCursor, listErr := ns.ListRecords(ctx.Context(), uid, opts)
	if listErr != nil {
		setShowToast(ctx, "error", webMsg(ctx, "error.load.notifications"))
		return ctx.SendString("")
	}

	params := partials.NotificationHistoryParams{
		Records:    records,
		NextCursor: nextCursor,
		Group:      partials.NormalizeNotifyHistoryGroup(ctx.Query("group")),
		Channel:    ctx.Query("channel"),
		RuleID:     ctx.Query("rule_id"),
		Channels:   notifyHistoryChannelFacets(ctx.Context()),
		RuleIDs:    notifyHistoryRuleFacets(ctx.Context()),
	}
	if params.Group == "" && opts.UnreadOnly {
		params.Group = "unread"
	}

	ctx.Type("html")
	return partials.NotificationsTable(ctx.Context(), params).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

// notifyHistoryListOptions maps History query params onto notify list filters.
func notifyHistoryListOptions(ctx fiber.Ctx) notifypkg.ListNotifyRecordsOptions {
	group := partials.NormalizeNotifyHistoryGroup(ctx.Query("group"))
	opts := notifypkg.ListNotifyRecordsOptions{
		Limit:   20,
		Cursor:  ctx.Query("cursor"),
		Channel: ctx.Query("channel"),
		RuleID:  ctx.Query("rule_id"),
	}
	if group == "unread" {
		opts.UnreadOnly = true
	}
	return opts
}

// notifyHistoryChannelFacets returns channel names for the History filter dropdown.
func notifyHistoryChannelFacets(ctx context.Context) []string {
	if store.Database == nil {
		return nil
	}
	channels, err := store.NotifyConfigStoreFromDB().ListNotifyChannels(ctx, store.ListNotifyChannelOptions{})
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(channels))
	for _, ch := range channels {
		names = append(names, ch.Name)
	}
	return names
}

// notifyHistoryRuleFacets returns rule ids for the History filter dropdown.
func notifyHistoryRuleFacets(ctx context.Context) []string {
	if store.Database == nil {
		return nil
	}
	rules, err := store.NotifyConfigStoreFromDB().ListNotifyRules(ctx, store.ListNotifyRuleOptions{})
	if err != nil {
		return nil
	}
	ids := make([]string, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.RuleID)
	}
	return ids
}

// retryConnectivityTest re-runs a channel connectivity probe for the named channel.
func retryConnectivityTest(ctx context.Context, ns notifypkg.NotifyRecords, uid, channelName string) error {
	ch, err := lookupNotifyChannelRawByName(ctx, channelName)
	if err != nil {
		return err
	}
	notifyMsg := notifypkg.Message{
		Title:    i18n.T(ctx, "notify.test.title"),
		Body:     i18n.T(ctx, "notify.test.body"),
		Priority: notifypkg.Low,
	}
	if err := notifypkg.SendToProtocol(ch.Protocol, ch.URI, notifyMsg); err != nil {
		if ns != nil {
			_, _ = ns.Record(ctx, uid, ch.Name, notifypkg.ConnectivityTestTemplateID, i18n.T(ctx, "notify.test.summary"), "failed", err.Error(), "", nil)
		}
		return err
	}
	if ns != nil {
		_, _ = ns.Record(ctx, uid, ch.Name, notifypkg.ConnectivityTestTemplateID, i18n.T(ctx, "notify.test.summary"), "success", "", "", nil)
	}
	return nil
}

// lookupNotifyChannelRawByName finds a notify channel by name and returns its raw URI.
func lookupNotifyChannelRawByName(ctx context.Context, name string) (model.NotifyChannel, error) {
	if store.Database == nil {
		return model.NotifyChannel{}, fmt.Errorf("channel %q not found", name)
	}
	channels, err := store.NotifyConfigStoreFromDB().ListNotifyChannels(ctx, store.ListNotifyChannelOptions{})
	if err != nil {
		return model.NotifyChannel{}, err
	}
	for _, ch := range channels {
		if ch.Name != name {
			continue
		}
		raw, err := store.NotifyConfigStoreFromDB().GetNotifyChannelRaw(ctx, ch.ID)
		if err != nil {
			return model.NotifyChannel{}, err
		}
		return raw, nil
	}
	return model.NotifyChannel{}, fmt.Errorf("channel %q not found", name)
}
