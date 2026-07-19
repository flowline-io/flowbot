package web

import (
	"context"
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
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notifySettingsWebserviceRules = []webservice.Rule{
	webservice.Get("/notifications/channels/list", notifyChannelsTable, route.WithNotAuth()),
	webservice.Get("/notifications/channels/new", notifyChannelNewForm, route.WithNotAuth()),
	webservice.Post("/notifications/channels", notifyChannelCreate, route.WithNotAuth()),
	webservice.Get("/notifications/channels/:id/edit", notifyChannelEditForm, route.WithNotAuth()),
	webservice.Put("/notifications/channels/:id", notifyChannelUpdate, route.WithNotAuth()),
	webservice.Delete("/notifications/channels/:id", notifyChannelDelete, route.WithNotAuth()),
	webservice.Post("/notifications/channels/:id/test", notifyChannelTest, route.WithNotAuth()),
	webservice.Get("/notifications/templates/list", notifyTemplatesTable, route.WithNotAuth()),
	webservice.Get("/notifications/templates/new", notifyTemplateNewForm, route.WithNotAuth()),
	webservice.Post("/notifications/templates", notifyTemplateCreate, route.WithNotAuth()),
	webservice.Get("/notifications/templates/:id/edit", notifyTemplateEditForm, route.WithNotAuth()),
	webservice.Put("/notifications/templates/:id", notifyTemplateUpdate, route.WithNotAuth()),
	webservice.Delete("/notifications/templates/:id", notifyTemplateDelete, route.WithNotAuth()),
	webservice.Get("/notifications/rules/list", notifyRulesTable, route.WithNotAuth()),
	webservice.Get("/notifications/rules/new", notifyRuleNewForm, route.WithNotAuth()),
	webservice.Post("/notifications/rules", notifyRuleCreate, route.WithNotAuth()),
	webservice.Get("/notifications/rules/:id/edit", notifyRuleEditForm, route.WithNotAuth()),
	webservice.Put("/notifications/rules/:id", notifyRuleUpdate, route.WithNotAuth()),
	webservice.Delete("/notifications/rules/:id", notifyRuleDelete, route.WithNotAuth()),
}

func notifySettingsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	tab := normalizeNotifySettingsTab(ctx.Query("tab"))
	ctx.Type("html")
	return pages.NotifySettingsPage(tab).Render(ctx.Context(), ctx.Response().BodyWriter())
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
	channels, err := store.Database.ListNotifyChannels(ctx.Context(), store.ListNotifyChannelOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load channels").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyChannelsTable(channels).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	protocols := getProtocolsList()
	ctx.Type("html")
	return partials.NotifyChannelForm(model.NotifyChannel{}, true, nil, protocols).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	name := ctx.FormValue("name")
	protocol := ctx.FormValue("protocol")
	uri := ctx.FormValue("uri")
	errs := validateChannelForm(name, protocol, uri)
	if len(errs) > 0 {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(model.NotifyChannel{Name: name, Protocol: protocol, URI: uri}, true, errs, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.Database.CreateNotifyChannel(ctx.Context(), name, protocol, uri)
	if err != nil {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(model.NotifyChannel{Name: name, Protocol: protocol, URI: uri}, true,
			notifyFormErrorsFromStore(err), protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Channel created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	if err := partials.NotifyChannelRow(ch).Render(ctx.Context(), ctx.Response().BodyWriter()); err != nil {
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
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	protocols := getProtocolsList()
	ctx.Type("html")
	return partials.NotifyChannelForm(ch, false, nil, protocols).
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
	name := ctx.FormValue("name")
	protocol := ctx.FormValue("protocol")
	uri := ctx.FormValue("uri")
	enabled := ctx.FormValue("enabled") == "on"
	// Empty URI keeps the existing secret; only validate name/protocol when URI is omitted.
	errs := validateChannelForm(name, protocol, uri)
	if uri == "" {
		delete(errs, "uri")
	}
	if len(errs) > 0 {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(model.NotifyChannel{ID: id, Name: name, Protocol: protocol}, false, errs, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.Database.UpdateNotifyChannel(ctx.Context(), id, name, protocol, uri, enabled); err != nil {
		if fieldErrs := mapNotifyChannelUniqueError(err); len(fieldErrs) > 0 {
			protocols := getProtocolsList()
			ctx.Type("html")
			return partials.NotifyChannelForm(model.NotifyChannel{ID: id, Name: name, Protocol: protocol}, false, fieldErrs, protocols).
				Render(ctx.Context(), ctx.Response().BodyWriter())
		}
		return storeError(ctx, err)
	}
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyChannelRow(ch).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.Database.DeleteNotifyChannel(ctx.Context(), id); err != nil {
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
	ch, err := store.Database.GetNotifyChannelRaw(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	uid := getUID(ctx)
	if uid == "" {
		uid = "system"
	}
	notifyMsg := notifypkg.Message{
		Title:    "Test Notification",
		Body:     "Connectivity test from Flowbot",
		Priority: notifypkg.Low,
	}
	if err := notifypkg.SendToProtocol(ch.Protocol, ch.URI, notifyMsg); err != nil {
		setShowToast(ctx, "error", "Connection failed: "+err.Error())
		ns := notifypkg.GetNotifyStore()
		if ns != nil {
			_, _ = ns.Record(ctx.Context(), uid, ch.Name, notifypkg.ConnectivityTestTemplateID, "Test connectivity", "failed", err.Error(), nil)
		}
		return ctx.SendString("")
	}
	setShowToast(ctx, "success", "Connection successful")
	ns := notifypkg.GetNotifyStore()
	if ns != nil {
		_, _ = ns.Record(ctx.Context(), uid, ch.Name, notifypkg.ConnectivityTestTemplateID, "Test connectivity", "success", "", nil)
	}
	return ctx.SendString("")
}

// ---------------------------------------------------------------------------
// Template handlers
// ---------------------------------------------------------------------------

func notifyTemplatesTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	templates, err := store.Database.ListNotifyTemplates(ctx.Context(), store.ListNotifyTemplateOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load templates").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyTemplatesTable(templates).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return partials.NotifyTemplateForm(model.NotifyTemplate{DefaultFormat: "markdown", OverridesJSON: "[]"}, true, nil).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	tmpl := parseTemplateForm(ctx)
	errs := validateTemplateForm(tmpl)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyTemplateForm(tmpl, true, errs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.Database.CreateNotifyTemplate(ctx.Context(), tmpl)
	if err != nil {
		ctx.Type("html")
		return partials.NotifyTemplateForm(tmpl, true, notifyFormErrorsFromStore(err)).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	reloadTemplateEngine(ctx.Context())
	row, err := store.Database.GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Template created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	if err := partials.NotifyTemplateRow(row).Render(ctx.Context(), ctx.Response().BodyWriter()); err != nil {
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
	tmpl, err := store.Database.GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyTemplateForm(tmpl, false, nil).
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
	errs := validateTemplateForm(tmpl)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyTemplateForm(tmpl, false, errs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.Database.UpdateNotifyTemplate(ctx.Context(), id, tmpl); err != nil {
		if fieldErrs := mapNotifyTemplateUniqueError(err); len(fieldErrs) > 0 {
			ctx.Type("html")
			return partials.NotifyTemplateForm(tmpl, false, fieldErrs).
				Render(ctx.Context(), ctx.Response().BodyWriter())
		}
		return storeError(ctx, err)
	}
	reloadTemplateEngine(ctx.Context())
	row, err := store.Database.GetNotifyTemplate(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyTemplateRow(row).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyTemplateDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.Database.DeleteNotifyTemplate(ctx.Context(), id); err != nil {
		return storeError(ctx, err)
	}
	reloadTemplateEngine(ctx.Context())
	return ctx.SendString("")
}

// ---------------------------------------------------------------------------
// Rule handlers
// ---------------------------------------------------------------------------

func notifyRulesTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rules, err := store.Database.ListNotifyRules(ctx.Context(), store.ListNotifyRuleOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load rules").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRulesTable(rules, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRuleForm(model.NotifyRule{Enabled: true, EventPattern: "*", ChannelPattern: "*"}, true, nil, templateIDs).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rule := parseRuleForm(ctx)
	templateIDs := getTemplateIDs()
	errs := validateRuleForm(rule)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyRuleForm(rule, true, errs, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.Database.CreateNotifyRule(ctx.Context(), rule)
	if err != nil {
		ctx.Type("html")
		return partials.NotifyRuleForm(rule, true, notifyFormErrorsFromStore(err), templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Rule created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	if err := partials.NotifyRuleRow(r, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter()); err != nil {
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
	rule, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRuleForm(rule, false, nil, templateIDs).
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
	errs := validateRuleForm(rule)
	if len(errs) > 0 {
		ctx.Type("html")
		return partials.NotifyRuleForm(rule, false, errs, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.Database.UpdateNotifyRule(ctx.Context(), id, rule); err != nil {
		if fieldErrs := mapNotifyRuleUniqueError(err); len(fieldErrs) > 0 {
			rule.ID = id
			ctx.Type("html")
			return partials.NotifyRuleForm(rule, false, fieldErrs, templateIDs).
				Render(ctx.Context(), ctx.Response().BodyWriter())
		}
		return storeError(ctx, err)
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyRuleRow(r, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.Database.DeleteNotifyRule(ctx.Context(), id); err != nil {
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
	return partials.EmptyState("Not found").Render(ctx.Context(), ctx.Response().BodyWriter())
}

func storeError(ctx fiber.Ctx, err error) error {
	flog.Error(fmt.Errorf("notify settings store: %w", err))
	ctx.Type("html")
	return partials.EmptyState("Operation failed").Render(ctx.Context(), ctx.Response().BodyWriter())
}

// notifyFormErrorsFromStore maps a store error to user-facing form field errors.
// Unique constraint violations become field messages; other errors are logged and
// returned as a generic save failure without leaking SQL details.
func notifyFormErrorsFromStore(err error) map[string]string {
	if fieldErrs := mapNotifyChannelUniqueError(err); len(fieldErrs) > 0 {
		return fieldErrs
	}
	if fieldErrs := mapNotifyTemplateUniqueError(err); len(fieldErrs) > 0 {
		return fieldErrs
	}
	if fieldErrs := mapNotifyRuleUniqueError(err); len(fieldErrs) > 0 {
		return fieldErrs
	}
	flog.Error(fmt.Errorf("notify settings save: %w", err))
	return map[string]string{"_save": "Failed to save"}
}

// mapNotifyChannelUniqueError maps unique constraint failures on notify channels.
func mapNotifyChannelUniqueError(err error) map[string]string {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "notify_channels_name_key") {
		return map[string]string{"name": "Name already exists"}
	}
	return nil
}

// mapNotifyTemplateUniqueError maps unique constraint failures on notify templates.
func mapNotifyTemplateUniqueError(err error) map[string]string {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "notify_templates_template_id_key") {
		return map[string]string{"template_id": "Template ID already exists"}
	}
	return nil
}

// mapNotifyRuleUniqueError maps unique constraint failures on notify rules.
func mapNotifyRuleUniqueError(err error) map[string]string {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "notify_rules_rule_id_key") {
		return map[string]string{"rule_id": "Rule ID already exists"}
	}
	return nil
}

func getProtocolsList() []string {
	protocols := []string{}
	for id := range notifypkg.List() {
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

func validateTemplateForm(tmpl model.NotifyTemplate) map[string]string {
	errs := map[string]string{}
	if tmpl.TemplateID == "" {
		errs["template_id"] = "Template ID is required"
	}
	if tmpl.Name == "" {
		errs["name"] = "Name is required"
	}
	if tmpl.DefaultFormat == "" {
		errs["default_format"] = "Format is required"
	}
	if tmpl.DefaultTemplate == "" {
		errs["default_template"] = "Template body is required"
	}
	var overrides []notifypkg.Override
	if err := sonic.Unmarshal([]byte(tmpl.OverridesJSON), &overrides); err != nil {
		errs["overrides_json"] = "Invalid JSON: " + err.Error()
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
		errs["default_template"] = "Template compile error: " + err.Error()
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

func validateChannelForm(name, protocol, uri string) map[string]string {
	errs := map[string]string{}
	if name == "" {
		errs["name"] = "Name is required"
	}
	if protocol == "" {
		errs["protocol"] = "Protocol is required"
	}
	if uri == "" {
		errs["uri"] = "URI is required"
	}
	return errs
}

func validateRuleForm(rule model.NotifyRule) map[string]string {
	errs := map[string]string{}
	if rule.Name == "" {
		errs["name"] = "Name is required"
	}
	if rule.RuleID == "" {
		errs["rule_id"] = "Rule ID is required"
	}
	if rule.EventPattern == "" {
		errs["event_pattern"] = "Event pattern is required"
	}
	if rule.ChannelPattern == "" {
		errs["channel_pattern"] = "Channel pattern is required"
	}
	if rule.Action == "" {
		errs["action"] = "Action is required"
	}
	if rule.Condition != "" {
		if err := notifyrules.ValidateCondition(rule.Condition); err != nil {
			errs["condition"] = err.Error()
		}
	}
	validateNotifyRuleParams(rule, &errs)
	return errs
}

func validateNotifyRuleParams(rule model.NotifyRule, errs *map[string]string) {
	switch rule.Action {
	case "throttle", "aggregate":
		// continue
	default:
		return
	}
	if rule.ParamsJSON == "" {
		(*errs)["window"] = "Window is required"
		if rule.Action == "throttle" {
			(*errs)["limit"] = "Limit is required"
		}
		return
	}
	var params map[string]any
	if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
		(*errs)["window"] = "Invalid parameters"
		return
	}
	switch rule.Action {
	case "throttle":
		validateThrottleParams(params, errs)
	case "aggregate":
		validateAggregateParams(params, errs)
	}
}

func validateThrottleParams(params map[string]any, errs *map[string]string) {
	if w, ok := params["window"].(string); !ok || w == "" {
		(*errs)["window"] = "Window is required"
	}
	if l, ok := params["limit"]; !ok {
		(*errs)["limit"] = "Limit is required"
	} else if v, ok := l.(float64); ok && v <= 0 {
		(*errs)["limit"] = "Limit must be > 0"
	} else if v, ok := l.(int); ok && v <= 0 {
		(*errs)["limit"] = "Limit must be > 0"
	}
}

func validateAggregateParams(params map[string]any, errs *map[string]string) {
	if w, ok := params["window"].(string); !ok || w == "" {
		(*errs)["window"] = "Window is required"
		return
	}
	if tid, ok := params["digest_template_id"].(string); ok && tid != "" {
		if eng := notifytmpl.GetEngine(); eng != nil && !eng.HasTemplate(tid) {
			(*errs)["digest_template_id"] = "Unknown template: " + tid
		}
	}
}

func reloadRulesEngine(ctx context.Context) {
	eng := notifyrules.GetEngine()
	if eng == nil {
		return
	}
	enabled := true
	rules, err := store.Database.ListNotifyRules(ctx, store.ListNotifyRuleOptions{Enabled: &enabled})
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
	rows, err := store.Database.ListNotifyTemplates(ctx, store.ListNotifyTemplateOptions{})
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
