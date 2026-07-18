package web

import (
	"context"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
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
	webservice.Get("/notify-settings", notifySettingsPage, route.WithNotAuth()),
	webservice.Get("/notify-settings/channels/list", notifyChannelsTable, route.WithNotAuth()),
	webservice.Get("/notify-settings/channels/new", notifyChannelNewForm, route.WithNotAuth()),
	webservice.Post("/notify-settings/channels", notifyChannelCreate, route.WithNotAuth()),
	webservice.Get("/notify-settings/channels/:id/edit", notifyChannelEditForm, route.WithNotAuth()),
	webservice.Put("/notify-settings/channels/:id", notifyChannelUpdate, route.WithNotAuth()),
	webservice.Delete("/notify-settings/channels/:id", notifyChannelDelete, route.WithNotAuth()),
	webservice.Post("/notify-settings/channels/:id/test", notifyChannelTest, route.WithNotAuth()),
	webservice.Get("/notify-settings/rules/list", notifyRulesTable, route.WithNotAuth()),
	webservice.Get("/notify-settings/rules/new", notifyRuleNewForm, route.WithNotAuth()),
	webservice.Post("/notify-settings/rules", notifyRuleCreate, route.WithNotAuth()),
	webservice.Get("/notify-settings/rules/:id/edit", notifyRuleEditForm, route.WithNotAuth()),
	webservice.Put("/notify-settings/rules/:id", notifyRuleUpdate, route.WithNotAuth()),
	webservice.Delete("/notify-settings/rules/:id", notifyRuleDelete, route.WithNotAuth()),
}

func notifySettingsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return pages.NotifySettingsPage().Render(ctx.Context(), ctx.Response().BodyWriter())
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
			map[string]string{"_save": err.Error()}, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Channel created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyChannelRow(ch).Render(ctx.Context(), ctx.Response().BodyWriter())
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
	return partials.NotifyRuleForm(model.NotifyRule{}, true, nil, templateIDs).
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
		return partials.NotifyRuleForm(rule, true, map[string]string{"_save": err.Error()}, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Rule created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyRuleRow(r, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter())
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

// showToastTrigger builds an HX-Trigger payload for the web UI toast system.
func showToastTrigger(toastType, message string) (string, error) {
	payload, err := sonic.Marshal(map[string]any{
		"showToast": map[string]string{
			"type":    toastType,
			"message": message,
		},
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// setShowToast sets the HX-Trigger header so HTMX can fire a showToast event.
func setShowToast(ctx fiber.Ctx, toastType, message string) {
	trigger, err := showToastTrigger(toastType, message)
	if err != nil {
		return
	}
	ctx.Set("HX-Trigger", trigger)
}

func notFound(ctx fiber.Ctx) error {
	ctx.Type("html")
	return partials.EmptyState("Not found").Render(ctx.Context(), ctx.Response().BodyWriter())
}

func storeError(ctx fiber.Ctx, err error) error {
	ctx.Type("html")
	return partials.EmptyState(err.Error()).Render(ctx.Context(), ctx.Response().BodyWriter())
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

func parseRuleForm(ctx fiber.Ctx) model.NotifyRule {
	prio, _ := strconv.Atoi(ctx.FormValue("priority"))
	enabled := ctx.FormValue("enabled") == "on"
	return model.NotifyRule{
		RuleID:         ctx.FormValue("rule_id"),
		Name:           ctx.FormValue("name"),
		Action:         ctx.FormValue("action"),
		EventPattern:   ctx.FormValue("event_pattern"),
		ChannelPattern: ctx.FormValue("channel_pattern"),
		Condition:      ctx.FormValue("condition"),
		Priority:       prio,
		ParamsJSON:     ctx.FormValue("params_json"),
		Enabled:        enabled,
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
	if rule.ParamsJSON == "" {
		return
	}
	var params map[string]any
	if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
		(*errs)["params_json"] = "Invalid JSON: " + err.Error()
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
		(*errs)["params_json"] = "Window is required"
	}
	if l, ok := params["limit"]; !ok {
		(*errs)["params_json"] = "Limit is required"
	} else if v, ok := l.(float64); ok && v <= 0 {
		(*errs)["params_json"] = "Limit must be > 0"
	}
}

func validateAggregateParams(params map[string]any, errs *map[string]string) {
	if w, ok := params["window"].(string); !ok || w == "" {
		(*errs)["params_json"] = "Window is required"
	}
	if tid, ok := params["digest_tpl_id"].(string); ok && tid != "" {
		if eng := notifytmpl.GetEngine(); eng != nil && !eng.HasTemplate(tid) {
			(*errs)["params_json"] = "Unknown template: " + tid
		}
	}
}

func reloadRulesEngine(ctx context.Context) {
	enabled := true
	rules, err := store.Database.ListNotifyRules(ctx, store.ListNotifyRuleOptions{Enabled: &enabled})
	if err != nil {
		return
	}
	configRules := make([]pkgconfig.NotifyRule, 0, len(rules))
	for _, r := range rules {
		var cond string
		if r.Condition != "" {
			cond = r.Condition
		}
		var params pkgconfig.NotifyRuleParams
		if r.ParamsJSON != "" {
			_ = sonic.Unmarshal([]byte(r.ParamsJSON), &params)
		}
		configRules = append(configRules, pkgconfig.NotifyRule{
			ID:     r.RuleID,
			Action: pkgconfig.NotifyRuleAction(r.Action),
			Match: pkgconfig.NotifyRuleMatch{
				Event:   r.EventPattern,
				Channel: r.ChannelPattern,
			},
			Condition: cond,
			Priority:  r.Priority,
			Params:    params,
		})
	}
	_ = notifyrules.GetEngine().LoadConfig(configRules)
}
