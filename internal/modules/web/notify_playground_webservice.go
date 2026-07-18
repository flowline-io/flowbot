package web

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notifyPlaygroundWebserviceRules = []webservice.Rule{
	webservice.Get("/notifications/playground", notifyPlaygroundForm, route.WithNotAuth()),
	webservice.Get("/notifications/playground/sample-payload", notifyPlaygroundSamplePayload, route.WithNotAuth()),
	webservice.Post("/notifications/playground/preview", notifyPlaygroundPreview, route.WithNotAuth()),
	webservice.Post("/notifications/playground/send", notifyPlaygroundSend, route.WithNotAuth()),
}

// playgroundRequest holds parsed Notifications playground form values.
type playgroundRequest struct {
	Mode           string
	ChannelID      int64
	TemplateID     string
	CustomTemplate string
	Format         string
	PayloadJSON    string
	Priority       string
	URL            string
	ChannelProto   string
}

func notifyPlaygroundForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	view, err := loadPlaygroundView(ctx.Context())
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load playground").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyPlaygroundSamplePayload(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	templateID := strings.TrimSpace(ctx.Query("template_id"))
	if templateID == "" {
		templateID = strings.TrimSpace(ctx.FormValue("template_id"))
	}
	payloadJSON := defaultPlaygroundPayloadJSON()
	if templateID != "" {
		if tmpl, ok := findPlaygroundTemplate(templateID); ok {
			if sample, err := notifytmpl.SamplePayloadJSON(tmpl); err == nil {
				payloadJSON = sample
			}
		}
	}
	ctx.Type("html")
	return partials.NotifyPlaygroundPayloadField(payloadJSON, "").Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyPlaygroundPreview(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	req := parsePlaygroundForm(ctx)
	view, err := loadPlaygroundView(ctx.Context())
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load playground").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	view.Form = playgroundFormFromRequest(req)
	errs := validatePlaygroundRequest(req)
	if len(errs) > 0 {
		view.Errors = errs
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := attachPlaygroundChannelProto(ctx.Context(), &req); err != nil {
		view.Errors = map[string]string{"channel_id": err.Error()}
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	rendered, err := renderPlaygroundMessage(req)
	if err != nil {
		view.Errors = map[string]string{"_form": err.Error()}
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	view.Result = &partials.NotifyPlaygroundResultParams{
		Title:   rendered.Title,
		Body:    rendered.Body,
		Format:  rendered.Format,
		Preview: true,
	}
	ctx.Type("html")
	return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyPlaygroundSend(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	req := parsePlaygroundForm(ctx)
	view, err := loadPlaygroundView(ctx.Context())
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load playground").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	view.Form = playgroundFormFromRequest(req)
	errs := validatePlaygroundRequest(req)
	if len(errs) > 0 {
		view.Errors = errs
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ch, err := store.Database.GetNotifyChannelRaw(ctx.Context(), req.ChannelID)
	if err != nil {
		view.Errors = map[string]string{"channel_id": "Channel not found"}
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if !ch.Enabled {
		view.Errors = map[string]string{"channel_id": "Channel is disabled"}
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	req.ChannelProto = ch.Protocol

	rendered, err := renderPlaygroundMessage(req)
	if err != nil {
		view.Errors = map[string]string{"_form": err.Error()}
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	msg := notifypkg.Message{
		Title:    rendered.Title,
		Body:     rendered.Body,
		Url:      strings.TrimSpace(req.URL),
		Priority: parsePlaygroundPriority(req.Priority),
	}
	uid := getUID(ctx)
	if uid == "" {
		uid = "system"
	}
	templateID := playgroundHistoryTemplateID(req)
	summary := rendered.Title
	if summary == "" {
		summary = "Playground send"
	}
	payload, _ := parsePlaygroundPayload(req.PayloadJSON)

	if err := notifypkg.SendToProtocol(ch.Protocol, ch.URI, msg); err != nil {
		ns := notifypkg.GetNotifyStore()
		if ns != nil {
			_, _ = ns.Record(ctx.Context(), uid, ch.Name, templateID, summary, "failed", err.Error(), payload)
		}
		setShowToast(ctx, "error", "Send failed: "+err.Error())
		view.Result = &partials.NotifyPlaygroundResultParams{
			Title:  rendered.Title,
			Body:   rendered.Body,
			Format: rendered.Format,
			Error:  err.Error(),
		}
		ctx.Type("html")
		return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ns := notifypkg.GetNotifyStore()
	if ns != nil {
		_, _ = ns.Record(ctx.Context(), uid, ch.Name, templateID, summary, "success", "", payload)
	}
	setShowToast(ctx, "success", "Notification sent to "+ch.Name)
	view.Result = &partials.NotifyPlaygroundResultParams{
		Title:   rendered.Title,
		Body:    rendered.Body,
		Format:  rendered.Format,
		Success: true,
	}
	ctx.Type("html")
	return partials.NotifyPlayground(view).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func loadPlaygroundView(ctx context.Context) (partials.NotifyPlaygroundParams, error) {
	enabled := true
	channels, err := store.Database.ListNotifyChannels(ctx, store.ListNotifyChannelOptions{Enabled: &enabled})
	if err != nil {
		return partials.NotifyPlaygroundParams{}, err
	}
	return partials.NotifyPlaygroundParams{
		Channels:  channels,
		Templates: getTemplates(),
		Form: partials.NotifyPlaygroundForm{
			Mode:        "template",
			Format:      "markdown",
			Priority:    "normal",
			PayloadJSON: defaultPlaygroundPayloadJSON(),
		},
	}, nil
}

func defaultPlaygroundPayloadJSON() string {
	return "{\n  \"summary\": \"playground\"\n}"
}

func findPlaygroundTemplate(id string) (notifypkg.Template, bool) {
	for _, tmpl := range getTemplates() {
		if tmpl.ID == id {
			return tmpl, true
		}
	}
	return notifypkg.Template{}, false
}

func parsePlaygroundForm(ctx fiber.Ctx) playgroundRequest {
	channelID, _ := strconv.ParseInt(ctx.FormValue("channel_id"), 10, 64)
	mode := ctx.FormValue("mode")
	if mode == "" {
		mode = "template"
	}
	format := ctx.FormValue("format")
	if format == "" {
		format = "markdown"
	}
	priority := ctx.FormValue("priority")
	if priority == "" {
		priority = "normal"
	}
	return playgroundRequest{
		Mode:           mode,
		ChannelID:      channelID,
		TemplateID:     strings.TrimSpace(ctx.FormValue("template_id")),
		CustomTemplate: ctx.FormValue("custom_template"),
		Format:         format,
		PayloadJSON:    ctx.FormValue("payload_json"),
		Priority:       priority,
		URL:            ctx.FormValue("url"),
	}
}

func playgroundFormFromRequest(req playgroundRequest) partials.NotifyPlaygroundForm {
	return partials.NotifyPlaygroundForm{
		Mode:           req.Mode,
		ChannelID:      req.ChannelID,
		TemplateID:     req.TemplateID,
		CustomTemplate: req.CustomTemplate,
		Format:         req.Format,
		PayloadJSON:    req.PayloadJSON,
		Priority:       req.Priority,
		URL:            req.URL,
	}
}

// validatePlaygroundRequest checks required playground fields before preview/send.
func validatePlaygroundRequest(req playgroundRequest) map[string]string {
	errs := map[string]string{}
	if req.ChannelID <= 0 {
		errs["channel_id"] = "Channel is required"
	}
	switch req.Mode {
	case "custom":
		if strings.TrimSpace(req.CustomTemplate) == "" {
			errs["custom_template"] = "Custom template is required"
		}
	default:
		if req.TemplateID == "" {
			errs["template_id"] = "Template is required"
		} else if !playgroundTemplateExists(req.TemplateID) {
			errs["template_id"] = "Unknown template"
		}
	}
	if _, err := parsePlaygroundPayload(req.PayloadJSON); err != nil {
		errs["payload_json"] = "Invalid JSON: " + err.Error()
	}
	return errs
}

func playgroundTemplateExists(id string) bool {
	if eng := notifytmpl.GetEngine(); eng != nil {
		return eng.HasTemplate(id)
	}
	return false
}

func parsePlaygroundPayload(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := sonic.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return map[string]any{}, nil
	}
	return payload, nil
}

func attachPlaygroundChannelProto(ctx context.Context, req *playgroundRequest) error {
	if req.ChannelID <= 0 {
		return fmt.Errorf("Channel is required")
	}
	ch, err := store.Database.GetNotifyChannel(ctx, req.ChannelID)
	if err != nil {
		return fmt.Errorf("Channel not found")
	}
	req.ChannelProto = ch.Protocol
	return nil
}

func renderPlaygroundMessage(req playgroundRequest) (*notifytmpl.RenderResult, error) {
	payload, err := parsePlaygroundPayload(req.PayloadJSON)
	if err != nil {
		return nil, err
	}
	if req.Mode == "custom" {
		return notifytmpl.RenderString(req.CustomTemplate, req.Format, payload)
	}
	eng := notifytmpl.GetEngine()
	if eng == nil {
		return nil, fmt.Errorf("template engine not initialized")
	}
	result, err := eng.Render(req.TemplateID, req.ChannelProto, payload)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("template %q not found", req.TemplateID)
	}
	return result, nil
}

func playgroundHistoryTemplateID(req playgroundRequest) string {
	if req.Mode == "custom" {
		return notifypkg.PlaygroundTemplateID
	}
	return req.TemplateID
}

func parsePlaygroundPriority(raw string) notifypkg.Priority {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low":
		return notifypkg.Low
	case "moderate":
		return notifypkg.Moderate
	case "high":
		return notifypkg.High
	case "emergency":
		return notifypkg.Emergency
	default:
		return notifypkg.Normal
	}
}
