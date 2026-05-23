package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/samber/oops"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/internal/platforms/tailchat"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/page"
	"github.com/flowline-io/flowbot/pkg/page/form"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	formRule "github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	pageRule "github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/flowline-io/flowbot/pkg/validate"
)

func handleRoutes(a *fiber.App, ctl *Controller) {
	// Webservice
	for _, bot := range module.List() {
		bot.Webservice(a)
	}

	// hub management plane
	a.Get("/hub/apps", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsRead, ctl.hubApps)))
	a.Get("/hub/apps/:name", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsRead, ctl.hubApp)))
	a.Get("/hub/apps/:name/status", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsStatus, ctl.hubAppStatus)))
	a.Get("/hub/apps/:name/logs", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsLogs, ctl.hubAppLogs)))
	a.Post("/hub/apps/:name/start", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsStart, ctl.hubAppStart)))
	a.Post("/hub/apps/:name/stop", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsStop, ctl.hubAppStop)))
	a.Post("/hub/apps/:name/restart", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsRestart, ctl.hubAppRestart)))
	a.Post("/hub/apps/:name/pull", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsPull, ctl.hubAppPull)))
	a.Post("/hub/apps/:name/update", route.Authorize(0, route.RequireScope(auth.ScopeHubAppsUpdate, ctl.hubAppUpdate)))
	a.Get("/hub/capabilities", route.Authorize(0, route.RequireScope(auth.ScopeHubCapabilitiesRead, ctl.hubCapabilities)))
	a.Get("/hub/capabilities/:type", route.Authorize(0, route.RequireScope(auth.ScopeHubCapabilitiesRead, ctl.hubCapability)))
	a.Get("/hub/health", route.Authorize(0, route.RequireScope(auth.ScopeHubHealthRead, ctl.hubHealth)))

	// common
	a.Get("/", func(_ fiber.Ctx) error { return nil })
	a.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	a.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	a.Get(healthcheck.StartupEndpoint, healthcheck.New())
	a.All("/oauth/:provider/:flag", ctl.storeOAuth)
	a.Get("/p/:id", ctl.getPage)
	// metrics - expose prometheus metrics for scraping
	a.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})))
	// form
	a.Post("/form", ctl.postForm)
	// page
	a.Get("/page/:id/:flag", ctl.renderPage)
	// agent
	a.Post("/agent", ctl.agentData)
	// platform
	a.All("/platform/:platform", ctl.platformCallback)
}

// handler

type Controller struct {
	driver         protocol.Driver
	tailchatDriver protocol.Driver
	auditor        audit.Auditor
}

func newController(driver protocol.Driver, cfg *config.Type, storeAdapter store.Adapter, auditor audit.Auditor) *Controller {
	return &Controller{
		driver:         driver,
		tailchatDriver: tailchat.NewDriver(cfg, storeAdapter),
		auditor:        auditor,
	}
}

func (*Controller) storeOAuth(ctx fiber.Ctx) error {
	name := ctx.Params("provider")
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(ctx.Context(), flag)
	if err != nil {
		return protocol.ErrFlagError.Wrap(err)
	}
	if store.ParameterIsExpired(p) {
		return protocol.ErrFlagExpired.New("oauth error")
	}

	params := types.KV(p.Params)
	uid, _ := params.String("uid")
	topic, _ := params.String("topic")
	if uid == "" {
		return protocol.ErrBadParam.New("uid empty")
	}

	// code -> token
	provider := newProvider(name)
	tk, err := provider.GetAccessToken(ctx)
	if err != nil {
		return protocol.ErrOAuthError.Wrap(err)
	}

	token, ok := tk["token"].(string)
	if !ok {
		return protocol.ErrBadParam.New("missing or invalid token in access token response")
	}

	// store
	extra := types.KV{}
	_ = extra.Scan(tk["extra"])
	err = store.Database.OAuthSet(ctx.Context(), gen.OAuth{
		UID:   uid,
		Topic: topic,
		Name:  name,
		Type:  name,
		Token: token,
		Extra: schema.JSON(extra),
	})
	if err != nil {
		return protocol.ErrOAuthError.Wrap(err)
	}

	return ctx.SendString("ok")
}

func (*Controller) getPage(ctx fiber.Ctx) error {
	id := ctx.Params("id")

	p, err := store.Database.PageGet(ctx.Context(), id)
	if err != nil {
		return protocol.ErrNotFound.Wrap(err)
	}

	sch := types.KV(p.Schema)
	title, _ := sch.String("title")
	if title == "" {
		title = "Page"
	}
	ctx.Set("Content-Type", "text/html")
	var comp app.UI
	switch p.Type {
	case string(schema.PageForm):
		f, _ := store.Database.FormGet(ctx.Context(), p.PageID)
		comp = page.RenderForm(p, f)
	case string(schema.PageTable):
		comp = page.RenderTable(p)
	case string(schema.PageHtml):
		comp = page.RenderHtml(p)
	default:
		return protocol.ErrBadRequest.New("error type")
	}

	return ctx.SendString(page.RenderComponent(title, comp))
}

func (*Controller) renderPage(ctx fiber.Ctx) error {
	pageRuleId := ctx.Params("id")
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(ctx.Context(), flag)
	if err != nil {
		return protocol.ErrFlagError.Wrap(err)
	}
	if store.ParameterIsExpired(p) {
		return protocol.ErrFlagExpired.New("page error")
	}

	args := types.KV{}
	// add query params
	queries := ctx.Queries()
	if len(queries) > 0 {
		for k, v := range queries {
			args[k] = v
		}
	}

	kv := types.KV(p.Params)
	platform, _ := kv.String("platform")
	topic, _ := kv.String("topic")
	uid, _ := kv.String("uid")

	typesCtx := types.Context{
		Platform:   platform,
		Topic:      topic,
		AsUser:     types.Uid(uid),
		PageRuleId: pageRuleId,
		TraceCtx:   ctx.Context(),
	}

	_, botHandler := module.FindRuleAndHandler[pageRule.Rule](pageRuleId, module.List())

	if botHandler == nil {
		return protocol.ErrNotFound.New("module not found")
	}

	html, err := botHandler.Page(typesCtx, flag, args)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return protocol.ErrNotFound.New("page not found")
		}
		return protocol.ErrBadParam.Wrap(err)
	}
	ctx.Set("Content-Type", "text/html")
	return ctx.SendString(html)
}

func (*Controller) postForm(ctx fiber.Ctx) error {
	formId := ctx.FormValue("x-form_id")
	if formId == "" {
		return protocol.ErrBadParam.New("form_id is required")
	}

	uid := ctx.FormValue("x-uid")
	if uid == "" {
		return protocol.ErrBadParam.New("uid is required")
	}

	topic := ctx.FormValue("x-topic")
	if topic == "" {
		return protocol.ErrBadParam.New("topic is required")
	}

	formData, err := store.Database.FormGet(ctx.Context(), formId)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if formData.State == int(schema.FormStateSubmitSuccess) || formData.State == int(schema.FormStateSubmitFailed) {
		return protocol.ErrBadParam.New("error form state")
	}

	formMsg, botHandler, err := getFormBotHandler(formData.Schema, formId)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	f, err := ctx.MultipartForm()
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	values := parseFormFieldValues(formMsg.Field, f.Value, ctx)

	formBuilder := form.NewBuilder(formMsg.Field)
	formBuilder.Data = values
	err = formBuilder.Validate()
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	botCtx := types.Context{
		Topic:      topic,
		AsUser:     types.Uid(uid),
		FormId:     formData.FormID,
		FormRuleId: formMsg.ID,
		TraceCtx:   ctx.Context(),
	}

	if botHandler != nil {
		if !botHandler.IsReady() {
			return protocol.ErrBadParam.Errorf("module %s unavailable", topic)
		}

		payload, err := botHandler.Form(botCtx, values)
		if err != nil {
			return protocol.ErrBadParam.Wrap(err)
		}

		stats.ModuleRunTotalCounter(stats.FormRuleset).Inc()

		if payload == nil {
			return ctx.JSON(protocol.NewSuccessResponse("empty message"))
		}

		return ctx.JSON(protocol.NewSuccessResponse(payload))
	}

	return ctx.JSON(protocol.NewSuccessResponse("ok"))
}

func parseFormFieldValues(fields []types.FormField, pf map[string][]string, ctx fiber.Ctx) types.KV {
	values := make(types.KV)
	for _, field := range fields {
		switch field.Type {
		case types.FormFieldCheckbox:
			value := pf[field.Key]
			switch field.ValueType {
			case types.FormFieldValueStringSlice:
				values[field.Key] = value
			case types.FormFieldValueInt64Slice:
				var tmp []int64
				for _, s := range value {
					i, _ := strconv.ParseInt(s, 10, 64)
					tmp = append(tmp, i)
				}
				values[field.Key] = tmp
			case types.FormFieldValueFloat64Slice:
				var tmp []float64
				for _, s := range value {
					i, _ := strconv.ParseFloat(s, 64)
					tmp = append(tmp, i)
				}
				values[field.Key] = tmp
			}
		default:
			value := ctx.FormValue(field.Key)
			switch field.ValueType {
			case types.FormFieldValueString:
				values[field.Key] = value
			case types.FormFieldValueBool:
				if value == "true" {
					values[field.Key] = true
				}
				if value == "false" {
					values[field.Key] = false
				}
			case types.FormFieldValueInt64:
				values[field.Key], _ = strconv.ParseInt(value, 10, 64)
			case types.FormFieldValueFloat64:
				values[field.Key], _ = strconv.ParseFloat(value, 64)
			}
		}
	}
	return values
}

func getFormBotHandler(sch schema.JSON, formId string) (*types.FormMsg, module.Handler, error) {
	d, err := sonic.Marshal(sch)
	if err != nil {
		return nil, nil, err
	}
	var formMsg types.FormMsg
	err = sonic.Unmarshal(d, &formMsg)
	if err != nil {
		return nil, nil, err
	}

	formSchema := types.KV(sch)
	formRuleId, ok := formSchema.String("id")
	if !ok {
		return &formMsg, nil, protocol.ErrBadParam.Errorf("form %s %s", formId, "error form rule id")
	}

	_, botHandler := module.FindRuleAndHandler[formRule.Rule](formRuleId, module.List())
	return &formMsg, botHandler, nil
}

func (*Controller) agentData(ctx fiber.Ctx) error {
	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(ctx.RequestCtx(), &r, true); err != nil {
		return protocol.ErrInternalServerError.Wrap(err)
	}
	// authorization
	uid, isValid := route.CheckAccessToken(route.GetAccessToken(&r))
	if !isValid {
		return protocol.ErrNotAuthorized.New("token not valided")
	}

	var data types.AgentData
	if err := sonic.Unmarshal(ctx.Body(), &data); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(data); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	result, err := agentAction(uid, data)
	if err != nil {
		flog.Error(err)
		return protocol.ErrBadParam.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func (c *Controller) platformCallback(ctx fiber.Ctx) error {
	platform := ctx.Params("platform")

	var err error
	switch platform {
	case tailchat.ID:
		err = c.tailchatDriver.HttpServer(ctx)
	case slack.ID:
		err = c.driver.HttpServer(ctx)
	default:
		return protocol.ErrNotFound.New("platform not found")
	}
	if err != nil {
		var e oops.OopsError
		if errors.As(err, &e) {
			return e
		}
		return err
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
