package web

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notifySettingsWebserviceRules = []webservice.Rule{
	webservice.Get("/notifications/channels/list", notifyChannelsTable),
	webservice.Get("/notifications/channels/new", notifyChannelNewForm),
	webservice.Post("/notifications/channels", notifyChannelCreate),
	webservice.Get("/notifications/channels/:id/edit", notifyChannelEditForm),
	webservice.Put("/notifications/channels/:id", notifyChannelUpdate),
	webservice.Delete("/notifications/channels/:id", notifyChannelDelete),
	webservice.Post("/notifications/channels/:id/test", notifyChannelTest),
	webservice.Post("/notifications/channels/:id/default", notifyChannelSetDefault),
	webservice.Get("/notifications/templates/list", notifyTemplatesTable),
	webservice.Get("/notifications/templates/new", notifyTemplateNewForm),
	webservice.Post("/notifications/templates", notifyTemplateCreate),
	webservice.Get("/notifications/templates/:id/edit", notifyTemplateEditForm),
	webservice.Put("/notifications/templates/:id", notifyTemplateUpdate),
	webservice.Delete("/notifications/templates/:id", notifyTemplateDelete),
	webservice.Post("/notifications/templates/:id/default", notifyTemplateSetDefault),
	webservice.Get("/notifications/rules/list", notifyRulesTable),
	webservice.Get("/notifications/rules/new", notifyRuleNewForm),
	webservice.Post("/notifications/rules", notifyRuleCreate),
	webservice.Get("/notifications/rules/:id/edit", notifyRuleEditForm),
	webservice.Put("/notifications/rules/:id", notifyRuleUpdate),
	webservice.Delete("/notifications/rules/:id", notifyRuleDelete),
}

func notifySettingsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	tab := normalizeNotifySettingsTab(ctx.Query("tab"))
	ctx.Type("html")
	return pages.NotifySettingsPage(ctx.Context(), tab, ctx.Query("channel"), ctx.Query("rule_id")).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

// normalizeNotifySettingsTab returns a known tab id or the default channels tab.
func normalizeNotifySettingsTab(tab string) string {
	switch tab {
	case "templates", "rules", "history", "playground":
		return tab
	case "notifications":
		// Legacy query value; History tab holds delivery records.
		return "history"
	default:
		return "channels"
	}
}

// ---------------------------------------------------------------------------
// Channel handlers
// ---------------------------------------------------------------------------

func notifyChannelsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	channels, err := store.NotifyConfigStoreFromDB().ListNotifyChannels(ctx.Context(), store.ListNotifyChannelOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.failed_load_channels")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyChannelsTable(ctx.Context(), channels, ctx.Query("highlight")).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	protocols := getProtocolsList()
	ctx.Type("html")
	return partials.NotifyChannelForm(ctx.Context(), model.NotifyChannel{}, true, nil, protocols).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	name := ctx.FormValue("name")
	protocol := ctx.FormValue("protocol")
	uri := ctx.FormValue("uri")
	errs := validateChannelForm(ctx, name, protocol, uri)
	if len(errs) > 0 {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(ctx.Context(), model.NotifyChannel{Name: name, Protocol: protocol, URI: uri}, true, errs, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.NotifyConfigStoreFromDB().CreateNotifyChannel(ctx.Context(), name, protocol, uri)
	if err != nil {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(ctx.Context(), model.NotifyChannel{Name: name, Protocol: protocol, URI: uri}, true,
			notifyFormErrorsFromStore(ctx, err), protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ch, err := store.NotifyConfigStoreFromDB().GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.channel_created_load_failed")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.channel_saved")
	if err := partials.NotifyChannelRow(ctx.Context(), ch, false).Render(ctx.Context(), ctx.Response().BodyWriter()); err != nil {
		return err
	}
	_, _ = ctx.Response().BodyWriter().Write([]byte(`<tr id="notify-channels-empty" hx-swap-oob="delete"></tr>`))
	return nil
}

func notifyChannelEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	ch, err := store.NotifyConfigStoreFromDB().GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	protocols := getProtocolsList()
	ctx.Type("html")
	return partials.NotifyChannelForm(ctx.Context(), ch, false, nil, protocols).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	existing, err := store.NotifyConfigStoreFromDB().GetNotifyChannelRaw(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	if notifypkg.IsSystemNotifyChannel(existing.Name) {
		setShowToastKey(ctx, "error", "toast.notify_settings.system_channel_readonly")
		ch, _ := store.NotifyConfigStoreFromDB().GetNotifyChannel(ctx.Context(), id)
		ctx.Type("html")
		return partials.NotifyChannelRow(ctx.Context(), ch, false).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	name := ctx.FormValue("name")
	protocol := ctx.FormValue("protocol")
	uri := ctx.FormValue("uri")
	enabled := ctx.FormValue("enabled") == "on"
	// Empty URI keeps the existing secret; only validate name/protocol when URI is omitted.
	errs := validateChannelForm(ctx, name, protocol, uri)
	if uri == "" {
		delete(errs, "uri")
	}
	if len(errs) > 0 {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(ctx.Context(), model.NotifyChannel{ID: id, Name: name, Protocol: protocol}, false, errs, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.NotifyConfigStoreFromDB().UpdateNotifyChannel(ctx.Context(), id, name, protocol, uri, enabled); err != nil {
		if fieldErrs := mapNotifyChannelUniqueError(ctx, err); len(fieldErrs) > 0 {
			protocols := getProtocolsList()
			ctx.Type("html")
			return partials.NotifyChannelForm(ctx.Context(), model.NotifyChannel{ID: id, Name: name, Protocol: protocol}, false, fieldErrs, protocols).
				Render(ctx.Context(), ctx.Response().BodyWriter())
		}
		return storeError(ctx, err)
	}
	ch, err := store.NotifyConfigStoreFromDB().GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.channel_saved")
	return partials.NotifyChannelRow(ctx.Context(), ch, false).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	existing, err := store.NotifyConfigStoreFromDB().GetNotifyChannelRaw(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	if notifypkg.IsSystemNotifyChannel(existing.Name) {
		setShowToastKey(ctx, "error", "toast.notify_settings.system_channel_no_delete")
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	if err := store.NotifyConfigStoreFromDB().DeleteNotifyChannel(ctx.Context(), id); err != nil {
		return storeError(ctx, err)
	}
	return ctx.SendString("")
}

func notifyChannelTest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	ch, err := store.NotifyConfigStoreFromDB().GetNotifyChannelRaw(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	uid := getUID(ctx)
	if uid == "" {
		uid = "system"
	}
	notifyMsg := notifypkg.Message{
		Title:    webMsg(ctx, "notify.test.title"),
		Body:     webMsg(ctx, "notify.test.body"),
		Priority: notifypkg.Low,
	}
	if err := notifypkg.SendToProtocol(ch.Protocol, ch.URI, notifyMsg); err != nil {
		setShowToast(ctx, "error", webMsgData(ctx, "toast.notify_settings.channel_test_failed", map[string]any{"Error": err.Error()}))
		ns := notifypkg.GetNotifyStore()
		if ns != nil {
			_, _ = ns.Record(ctx.Context(), uid, ch.Name, notifypkg.ConnectivityTestTemplateID, webMsg(ctx, "notify.test.summary"), "failed", err.Error(), "", nil)
		}
		return ctx.SendString("")
	}
	setShowToastKey(ctx, "success", "toast.notify_settings.channel_test_succeeded")
	ns := notifypkg.GetNotifyStore()
	if ns != nil {
		_, _ = ns.Record(ctx.Context(), uid, ch.Name, notifypkg.ConnectivityTestTemplateID, webMsg(ctx, "notify.test.summary"), "success", "", "", nil)
	}
	return ctx.SendString("")
}

func notifyChannelSetDefault(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.NotifyConfigStoreFromDB().SetDefaultNotifyChannel(ctx.Context(), id); err != nil {
		switch {
		case errors.Is(err, types.ErrNotFound):
			return toastErrorKey(ctx, "toast.notify_settings.channel_not_found")
		case errors.Is(err, types.ErrInvalidArgument):
			return toastErrorKey(ctx, "toast.notify_settings.channel_must_be_enabled")
		default:
			return toastErrorKey(ctx, "toast.notify_settings.default_channel_failed")
		}
	}
	channels, err := store.NotifyConfigStoreFromDB().ListNotifyChannels(ctx.Context(), store.ListNotifyChannelOptions{})
	if err != nil {
		return storeError(ctx, err)
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.default_channel_updated")
	return partials.NotifyChannelsTable(ctx.Context(), channels, "").Render(ctx.Context(), ctx.Response().BodyWriter())
}

// ---------------------------------------------------------------------------
// Template handlers
// ---------------------------------------------------------------------------

func notifyTemplatesTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	templates, err := store.NotifyConfigStoreFromDB().ListNotifyTemplates(ctx.Context(), store.ListNotifyTemplateOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.failed_load_templates")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyTemplatesTable(ctx.Context(), templates).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return partials.NotifyTemplateForm(ctx.Context(), model.NotifyTemplate{DefaultFormat: "markdown", OverridesJSON: "[]"}, true, nil).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	tmpl := parseTemplateForm(ctx)
	errs := validateTemplateForm(ctx, tmpl)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyTemplateForm(ctx.Context(), tmpl, true, errs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.NotifyConfigStoreFromDB().CreateNotifyTemplate(ctx.Context(), tmpl)
	if err != nil {
		ctx.Type("html")
		return partials.NotifyTemplateForm(ctx.Context(), tmpl, true, notifyFormErrorsFromStore(ctx, err)).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	reloadTemplateEngine(ctx.Context())
	row, err := store.NotifyConfigStoreFromDB().GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.template_created_load_failed")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.template_saved")
	if err := partials.NotifyTemplateRow(ctx.Context(), row).Render(ctx.Context(), ctx.Response().BodyWriter()); err != nil {
		return err
	}
	_, _ = ctx.Response().BodyWriter().Write([]byte(`<tr id="notify-templates-empty" hx-swap-oob="delete"></tr>`))
	return nil
}

func notifyTemplateEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	tmpl, err := store.NotifyConfigStoreFromDB().GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyTemplateForm(ctx.Context(), tmpl, false, nil).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	tmpl := parseTemplateForm(ctx)
	tmpl.ID = id
	errs := validateTemplateForm(ctx, tmpl)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyTemplateForm(ctx.Context(), tmpl, false, errs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.NotifyConfigStoreFromDB().UpdateNotifyTemplate(ctx.Context(), id, tmpl); err != nil {
		if fieldErrs := mapNotifyTemplateUniqueError(ctx, err); len(fieldErrs) > 0 {
			ctx.Type("html")
			return partials.NotifyTemplateForm(ctx.Context(), tmpl, false, fieldErrs).
				Render(ctx.Context(), ctx.Response().BodyWriter())
		}
		return storeError(ctx, err)
	}
	reloadTemplateEngine(ctx.Context())
	row, err := store.NotifyConfigStoreFromDB().GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.template_saved")
	return partials.NotifyTemplateRow(ctx.Context(), row).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.NotifyConfigStoreFromDB().DeleteNotifyTemplate(ctx.Context(), id); err != nil {
		return storeError(ctx, err)
	}
	reloadTemplateEngine(ctx.Context())
	return ctx.SendString("")
}

func notifyTemplateSetDefault(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	tmpl, err := store.NotifyConfigStoreFromDB().GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	if !notifypkg.TemplateReferencesSummary(tmpl.DefaultTemplate, tmpl.OverridesJSON) {
		return toastErrorKey(ctx, "toast.notify_settings.template_summary_required")
	}
	if err := store.NotifyConfigStoreFromDB().SetDefaultNotifyTemplate(ctx.Context(), id); err != nil {
		return toastErrorKey(ctx, "toast.notify_settings.default_template_failed")
	}
	templates, err := store.NotifyConfigStoreFromDB().ListNotifyTemplates(ctx.Context(), store.ListNotifyTemplateOptions{})
	if err != nil {
		return storeError(ctx, err)
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.default_template_updated")
	return partials.NotifyTemplatesTable(ctx.Context(), templates).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// ---------------------------------------------------------------------------
// Rule handlers
// ---------------------------------------------------------------------------

func notifyRulesTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rules, err := store.NotifyConfigStoreFromDB().ListNotifyRules(ctx.Context(), store.ListNotifyRuleOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.failed_load_rules")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRulesTable(ctx.Context(), rules, templateIDs, ctx.Query("highlight")).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRuleForm(ctx.Context(), model.NotifyRule{Enabled: true, EventPattern: "*", ChannelPattern: "*"}, true, nil, templateIDs).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rule := parseRuleForm(ctx)
	templateIDs := getTemplateIDs()
	errs := validateRuleForm(ctx, rule)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyRuleForm(ctx.Context(), rule, true, errs, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.NotifyConfigStoreFromDB().CreateNotifyRule(ctx.Context(), rule)
	if err != nil {
		ctx.Type("html")
		return partials.NotifyRuleForm(ctx.Context(), rule, true, notifyFormErrorsFromStore(ctx, err), templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.NotifyConfigStoreFromDB().GetNotifyRule(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.rule_created_load_failed")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.rule_saved")
	if err := partials.NotifyRuleRow(ctx.Context(), r, templateIDs, false).Render(ctx.Context(), ctx.Response().BodyWriter()); err != nil {
		return err
	}
	_, _ = ctx.Response().BodyWriter().Write([]byte(`<tr id="notify-rules-empty" hx-swap-oob="delete"></tr>`))
	return nil
}

func notifyRuleEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	rule, err := store.NotifyConfigStoreFromDB().GetNotifyRule(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRuleForm(ctx.Context(), rule, false, nil, templateIDs).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	rule := parseRuleForm(ctx)
	templateIDs := getTemplateIDs()
	errs := validateRuleForm(ctx, rule)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyRuleForm(ctx.Context(), rule, false, errs, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.NotifyConfigStoreFromDB().UpdateNotifyRule(ctx.Context(), id, rule); err != nil {
		if fieldErrs := mapNotifyRuleUniqueError(ctx, err); len(fieldErrs) > 0 {
			rule.ID = id
			ctx.Type("html")
			return partials.NotifyRuleForm(ctx.Context(), rule, false, fieldErrs, templateIDs).
				Render(ctx.Context(), ctx.Response().BodyWriter())
		}
		return storeError(ctx, err)
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.NotifyConfigStoreFromDB().GetNotifyRule(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.notify_settings.rule_saved")
	return partials.NotifyRuleRow(ctx.Context(), r, templateIDs, false).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.NotifyConfigStoreFromDB().DeleteNotifyRule(ctx.Context(), id); err != nil {
		return storeError(ctx, err)
	}
	reloadRulesEngine(ctx.Context())
	return ctx.SendString("")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseID(ctx fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return 0, err
	}
	return id, nil
}

func notFound(ctx fiber.Ctx) error {
	ctx.Type("html")
	return partials.EmptyState(webMsg(ctx, "empty.not_found")).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func storeError(ctx fiber.Ctx, err error) error {
	flog.Error(fmt.Errorf("notify settings store: %w", err))
	ctx.Type("html")
	return partials.EmptyState(webMsg(ctx, "empty.operation_failed")).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// notifyFormErrorsFromStore maps a store error to user-facing form field errors.
// Unique constraint violations become field messages; other errors are logged and
// returned as a generic save failure without leaking SQL details.
func notifyFormErrorsFromStore(c fiber.Ctx, err error) map[string]string {
	if fieldErrs := mapNotifyChannelUniqueError(c, err); len(fieldErrs) > 0 {
		return fieldErrs
	}
	if fieldErrs := mapNotifyTemplateUniqueError(c, err); len(fieldErrs) > 0 {
		return fieldErrs
	}
	if fieldErrs := mapNotifyRuleUniqueError(c, err); len(fieldErrs) > 0 {
		return fieldErrs
	}
	flog.Error(fmt.Errorf("notify settings save: %w", err))
	return map[string]string{"_save": webMsg(c, "error.validation.save_failed")}
}

// mapNotifyChannelUniqueError maps unique constraint failures on notify channels.
func mapNotifyChannelUniqueError(c fiber.Ctx, err error) map[string]string {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "notify_channels_name_key") {
		return map[string]string{"name": webMsg(c, "error.validation.name_exists")}
	}
	return nil
}

// mapNotifyTemplateUniqueError maps unique constraint failures on notify templates.
func mapNotifyTemplateUniqueError(c fiber.Ctx, err error) map[string]string {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "notify_templates_template_id_key") {
		return map[string]string{"template_id": webMsg(c, "error.validation.template_id_exists")}
	}
	return nil
}

// mapNotifyRuleUniqueError maps unique constraint failures on notify rules.
func mapNotifyRuleUniqueError(c fiber.Ctx, err error) map[string]string {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "notify_rules_rule_id_key") || strings.Contains(err.Error(), "notify_rules.rule_id") {
		return map[string]string{"rule_id": webMsg(c, "error.validation.rule_id_exists")}
	}
	return nil
}

func getProtocolsList() []string {
	list := notifypkg.List()
	protocols := make([]string, 0, len(list))
	for id := range list {
		protocols = append(protocols, id)
	}
	return protocols
}

func getTemplateIDs() []string {
	if eng := notifytmpl.GetEngine(); eng != nil {
		return eng.ListTemplateIDs()
	}
	return []string{}
}

// getTemplates returns notify template manifests from the engine.
func getTemplates() []notifypkg.Template {
	if eng := notifytmpl.GetEngine(); eng != nil {
		return eng.ListTemplates()
	}
	return []notifypkg.Template{}
}

func parseTemplateForm(ctx fiber.Ctx) model.NotifyTemplate {
	overridesJSON := strings.TrimSpace(ctx.FormValue("overrides_json"))
	if overridesJSON == "" {
		overridesJSON = "[]"
	}
	return model.NotifyTemplate{
		TemplateID:      strings.TrimSpace(ctx.FormValue("template_id")),
		Name:            strings.TrimSpace(ctx.FormValue("name")),
		Description:     strings.TrimSpace(ctx.FormValue("description")),
		DefaultFormat:   strings.TrimSpace(ctx.FormValue("default_format")),
		DefaultTemplate: ctx.FormValue("default_template"),
		OverridesJSON:   overridesJSON,
	}
}

func validateTemplateForm(c fiber.Ctx, tmpl model.NotifyTemplate) map[string]string {
	errs := map[string]string{}
	if tmpl.TemplateID == "" {
		errs["template_id"] = webMsg(c, "error.validation.template_id_required")
	}
	if tmpl.Name == "" {
		errs["name"] = webMsg(c, "error.validation.name_required")
	}
	if tmpl.DefaultFormat == "" {
		errs["default_format"] = webMsg(c, "error.validation.format_required")
	}
	if tmpl.DefaultTemplate == "" {
		errs["default_template"] = webMsg(c, "error.validation.template_body_required")
	}
	var overrides []notifypkg.Override
	if err := sonic.Unmarshal([]byte(tmpl.OverridesJSON), &overrides); err != nil {
		errs["overrides_json"] = webMsgData(c, "error.validation.invalid_json_detail", map[string]any{"Error": err.Error()})
		return errs
	}
	if len(errs) > 0 {
		return errs
	}
	engine := notifytmpl.New()
	if err := engine.LoadConfig([]notifypkg.Template{{
		ID:              tmpl.TemplateID,
		Name:            tmpl.Name,
		Description:     tmpl.Description,
		DefaultFormat:   tmpl.DefaultFormat,
		DefaultTemplate: tmpl.DefaultTemplate,
		Overrides:       overrides,
	}}); err != nil {
		errs["default_template"] = webMsgData(c, "error.validation.template_compile", map[string]any{"Error": err.Error()})
	}
	return errs
}

func parseRuleForm(ctx fiber.Ctx) model.NotifyRule {
	prio, _ := strconv.Atoi(ctx.FormValue("priority"))
	enabled := ctx.FormValue("enabled") == "on"
	action := ctx.FormValue("action")
	return model.NotifyRule{
		RuleID:         ctx.FormValue("rule_id"),
		Name:           ctx.FormValue("name"),
		Action:         action,
		EventPattern:   ctx.FormValue("event_pattern"),
		ChannelPattern: ctx.FormValue("channel_pattern"),
		Condition:      ctx.FormValue("condition"),
		Priority:       prio,
		ParamsJSON:     buildRuleParamsJSON(action, ctx),
		Enabled:        enabled,
	}
}

// buildRuleParamsJSON builds ParamsJSON from structured action form fields.
func buildRuleParamsJSON(action string, ctx fiber.Ctx) string {
	switch action {
	case "throttle":
		limit, _ := strconv.Atoi(strings.TrimSpace(ctx.FormValue("param_limit")))
		payload := map[string]any{
			"window": strings.TrimSpace(ctx.FormValue("param_window")),
			"limit":  limit,
		}
		s, err := sonic.MarshalString(payload)
		if err != nil {
			return ""
		}
		return s
	case "aggregate":
		payload := map[string]any{
			"window": strings.TrimSpace(ctx.FormValue("param_window")),
		}
		if tid := strings.TrimSpace(ctx.FormValue("param_digest_template_id")); tid != "" {
			payload["digest_template_id"] = tid
		}
		if ctx.FormValue("param_delayed_send") == "on" {
			payload["delayed_send"] = true
		}
		s, err := sonic.MarshalString(payload)
		if err != nil {
			return ""
		}
		return s
	default:
		return ""
	}
}

func validateChannelForm(c fiber.Ctx, name, protocol, uri string) map[string]string {
	errs := map[string]string{}
	if name == "" {
		errs["name"] = webMsg(c, "error.validation.name_required")
	}
	if protocol == "" {
		errs["protocol"] = webMsg(c, "error.validation.protocol_required")
	}
	if uri == "" {
		errs["uri"] = webMsg(c, "error.validation.uri_required")
	}
	return errs
}

func validateRuleForm(c fiber.Ctx, rule model.NotifyRule) map[string]string {
	errs := map[string]string{}
	if rule.Name == "" {
		errs["name"] = webMsg(c, "error.validation.name_required")
	}
	if rule.RuleID == "" {
		errs["rule_id"] = webMsg(c, "error.validation.rule_id_required")
	}
	if rule.EventPattern == "" {
		errs["event_pattern"] = webMsg(c, "error.validation.event_pattern_required")
	}
	if rule.ChannelPattern == "" {
		errs["channel_pattern"] = webMsg(c, "error.validation.channel_pattern_required")
	}
	if rule.Action == "" {
		errs["action"] = webMsg(c, "error.validation.action_required")
	}
	if rule.Condition != "" {
		if err := notifyrules.ValidateCondition(rule.Condition); err != nil {
			errs["condition"] = webMsg(c, "error.validation.invalid_condition")
		}
	}
	validateNotifyRuleParams(c, rule, &errs)
	return errs
}

func validateNotifyRuleParams(c fiber.Ctx, rule model.NotifyRule, errs *map[string]string) {
	switch rule.Action {
	case "throttle", "aggregate":
		// continue
	default:
		return
	}
	if rule.ParamsJSON == "" {
		(*errs)["window"] = webMsg(c, "error.validation.window_required")
		if rule.Action == "throttle" {
			(*errs)["limit"] = webMsg(c, "error.validation.limit_required")
		}
		return
	}
	var params map[string]any
	if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
		(*errs)["window"] = webMsg(c, "error.validation.invalid_params")
		return
	}
	switch rule.Action {
	case "throttle":
		validateThrottleParams(c, params, errs)
	case "aggregate":
		validateAggregateParams(c, params, errs)
	}
}

func validateThrottleParams(c fiber.Ctx, params map[string]any, errs *map[string]string) {
	if w, ok := params["window"].(string); !ok || w == "" {
		(*errs)["window"] = webMsg(c, "error.validation.window_required")
	}
	if l, ok := params["limit"]; !ok {
		(*errs)["limit"] = webMsg(c, "error.validation.limit_required")
	} else if v, ok := l.(float64); ok && v <= 0 {
		(*errs)["limit"] = webMsg(c, "error.validation.limit_positive")
	} else if v, ok := l.(int); ok && v <= 0 {
		(*errs)["limit"] = webMsg(c, "error.validation.limit_positive")
	}
}

func validateAggregateParams(c fiber.Ctx, params map[string]any, errs *map[string]string) {
	if w, ok := params["window"].(string); !ok || w == "" {
		(*errs)["window"] = webMsg(c, "error.validation.window_required")
		return
	}
	if tid, ok := params["digest_template_id"].(string); ok && tid != "" {
		if eng := notifytmpl.GetEngine(); eng != nil && !eng.HasTemplate(tid) {
			(*errs)["digest_template_id"] = webMsgData(c, "error.validation.unknown_template", map[string]any{"ID": tid})
		}
	}
}

func reloadRulesEngine(ctx context.Context) {
	eng := notifyrules.GetEngine()
	if eng == nil {
		return
	}
	enabled := true
	rules, err := store.NotifyConfigStoreFromDB().ListNotifyRules(ctx, store.ListNotifyRuleOptions{Enabled: &enabled})
	if err != nil {
		flog.Warn("reload notify rules: list failed: %v", err)
		return
	}
	manifestRules := make([]notifypkg.Rule, 0, len(rules))
	for _, r := range rules {
		var cond string
		if r.Condition != "" {
			cond = r.Condition
		}
		var params notifypkg.RuleParams
		if r.ParamsJSON != "" {
			if err := sonic.Unmarshal([]byte(r.ParamsJSON), &params); err != nil {
				flog.Warn("reload notify rules: skip %s: invalid params JSON: %v", r.RuleID, err)
				continue
			}
		}
		manifestRules = append(manifestRules, notifypkg.Rule{
			ID:     r.RuleID,
			Action: notifypkg.RuleAction(r.Action),
			Match: notifypkg.RuleMatch{
				Event:   r.EventPattern,
				Channel: r.ChannelPattern,
			},
			Condition: cond,
			Priority:  r.Priority,
			Params:    params,
		})
	}
	if err := eng.LoadConfig(manifestRules); err != nil {
		flog.Warn("reload notify rules: LoadConfig failed: %v", err)
	}
}

func reloadTemplateEngine(ctx context.Context) {
	rows, err := store.NotifyConfigStoreFromDB().ListNotifyTemplates(ctx, store.ListNotifyTemplateOptions{})
	if err != nil {
		flog.Warn("reload notify templates: list failed: %v", err)
		return
	}
	templates := make([]notifypkg.Template, 0, len(rows))
	for _, row := range rows {
		var overrides []notifypkg.Override
		if row.OverridesJSON != "" && row.OverridesJSON != "[]" {
			if err := sonic.Unmarshal([]byte(row.OverridesJSON), &overrides); err != nil {
				flog.Warn("reload notify templates: template %s has invalid overrides, using empty: %v", row.TemplateID, err)
				overrides = nil
			}
		}
		templates = append(templates, notifypkg.Template{
			ID:              row.TemplateID,
			Name:            row.Name,
			Description:     row.Description,
			DefaultFormat:   row.DefaultFormat,
			DefaultTemplate: row.DefaultTemplate,
			Overrides:       overrides,
		})
	}
	if err := notifytmpl.Init(templates); err != nil {
		flog.Warn("reload notify templates: Init failed: %v", err)
	}
}
