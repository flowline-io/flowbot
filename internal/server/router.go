package server

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/internal/platforms/tailchat"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/page"
	"github.com/flowline-io/flowbot/pkg/page/form"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	formRule "github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	pageRule "github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/gofiber/fiber/v2"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/samber/oops"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"gorm.io/gorm"
)

func handleRoutes(a *fiber.App, ctl *Controller) {
	// Webservice
	for _, bot := range bots.List() {
		bot.Webservice(a)
	}

	// common
	a.Get("/", func(c *fiber.Ctx) error { return nil })
	a.All("/oauth/:provider/:flag", ctl.storeOAuth)
	a.Get("/p/:id", ctl.getPage)
	// form
	a.Post("/form", ctl.postForm)
	// page
	a.Get("/page/:id/:flag", ctl.renderPage)
	// agent
	a.Post("/agent", ctl.agentData)
	// webhook
	a.All("/webhook/:flag", ctl.doWebhook)
	// platform
	a.All("/chatbot/:platform", ctl.platformCallback)
}

// handler

type Controller struct {
	driver protocol.Driver
}

func newController(driver protocol.Driver) *Controller {
	return &Controller{
		driver: driver,
	}
}

func (c *Controller) storeOAuth(ctx *fiber.Ctx) error {
	name := ctx.Params("provider")
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(flag)
	if err != nil {
		return protocol.ErrFlagError.Wrap(err)
	}
	if p.IsExpired() {
		return protocol.ErrFlagExpired.New("oauth error")
	}

	uid, _ := types.KV(p.Params).String("uid")
	topic, _ := types.KV(p.Params).String("topic")
	if uid == "" {
		return protocol.ErrBadParam.New("uid empty")
	}

	// code -> token
	provider := newProvider(name)
	tk, err := provider.GetAccessToken(ctx)
	if err != nil {
		return protocol.ErrOAuthError.Wrap(err)
	}

	// store
	extra := types.KV{}
	_ = extra.Scan(tk["extra"])
	err = store.Database.OAuthSet(model.OAuth{
		UID:   uid,
		Topic: topic,
		Name:  name,
		Type:  name,
		Token: tk["token"].(string),
		Extra: model.JSON(extra),
	})
	if err != nil {
		return protocol.ErrOAuthError.Wrap(err)
	}

	return ctx.SendString("ok")
}

func (c *Controller) getPage(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	p, err := store.Database.PageGet(id)
	if err != nil {
		return protocol.ErrNotFound.Wrap(err)
	}

	title, _ := types.KV(p.Schema).String("title")
	if title == "" {
		title = "Page"
	}
	ctx.Set("Content-Type", "text/html")
	var comp app.UI
	switch p.Type {
	case model.PageForm:
		f, _ := store.Database.FormGet(p.PageID)
		comp = page.RenderForm(p, f)
	case model.PageTable:
		comp = page.RenderTable(p)
	case model.PageHtml:
		comp = page.RenderHtml(p)
	case model.PageChart:
		d, err := sonic.Marshal(p.Schema)
		if err != nil {
			return protocol.ErrBadParam.Wrap(err)
		}
		var msg types.ChartMsg
		err = sonic.Unmarshal(d, &msg)
		if err != nil {
			return protocol.ErrBadParam.Wrap(err)
		}

		line := charts.NewLine()
		line.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
			Title:    msg.Title,
			Subtitle: msg.SubTitle,
		}), charts.WithInitializationOpts(opts.Initialization{PageTitle: "Chart"}))

		var lineData []opts.LineData
		for _, i := range msg.Series {
			lineData = append(lineData, opts.LineData{Value: i})
		}

		line.SetXAxis(msg.XAxis).AddSeries("Chart", lineData)

		buf := bytes.NewBuffer(nil)
		_ = line.Render(buf)
		return ctx.Send(buf.Bytes())
	default:
		return protocol.ErrBadRequest.New("error type")
	}

	return ctx.SendString(page.RenderComponent(title, comp))
}

func (c *Controller) renderPage(ctx *fiber.Ctx) error {
	pageRuleId := ctx.Params("id")
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(flag)
	if err != nil {
		return protocol.ErrFlagError.Wrap(err)
	}
	if p.IsExpired() {
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
	}

	_, botHandler := bots.FindRuleAndHandler[pageRule.Rule](pageRuleId, bots.List())

	if botHandler == nil {
		return protocol.ErrNotFound.New("bot not found")
	}

	html, err := botHandler.Page(typesCtx, flag, args)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return protocol.ErrNotFound.New("page not found")
		}
		return protocol.ErrBadParam.Wrap(err)
	}
	ctx.Set("Content-Type", "text/html")
	return ctx.SendString(html)
}

func (c *Controller) postForm(ctx *fiber.Ctx) error {
	formId := ctx.FormValue("x-form_id")
	uid := ctx.FormValue("x-uid")
	topic := ctx.FormValue("x-topic")

	formData, err := store.Database.FormGet(formId)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if formData.State == model.FormStateSubmitSuccess || formData.State == model.FormStateSubmitFailed {
		return protocol.ErrBadParam.New("error form state")
	}

	values := make(types.KV)
	d, err := sonic.Marshal(formData.Schema)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	var formMsg types.FormMsg
	err = sonic.Unmarshal(d, &formMsg)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	f, err := ctx.MultipartForm()
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	pf := f.Value
	for _, field := range formMsg.Field {
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
	}

	// user auth record

	// get bot handler
	formRuleId, ok := types.KV(formData.Schema).String("id")
	if !ok {
		return protocol.ErrBadParam.Errorf("form %s %s", formId, "error form rule id")
	}

	_, botHandler := bots.FindRuleAndHandler[formRule.Rule](formRuleId, bots.List())

	if botHandler != nil {
		if !botHandler.IsReady() {
			return protocol.ErrBadParam.Errorf("bot %s unavailable", topic)
		}

		// form message
		payload, err := botHandler.Form(botCtx, values)
		if err != nil {
			return protocol.ErrBadParam.Wrap(err)
		}

		// stats
		stats.BotRunTotalCounter(stats.FormRuleset).Inc()

		if payload == nil {
			return ctx.JSON(protocol.NewSuccessResponse("empty message"))
		}

		return ctx.JSON(protocol.NewSuccessResponse(payload))
	}

	return ctx.JSON(protocol.NewSuccessResponse("ok"))
}

func (c *Controller) agentData(ctx *fiber.Ctx) error {
	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(ctx.Context(), &r, true); err != nil {
		return protocol.ErrInternalServerError.Wrap(err)
	}
	// authorization
	uid, isValid := route.CheckAccessToken(route.GetAccessToken(&r))
	if !isValid {
		return protocol.ErrNotAuthorized.New("token not valided")
	}

	var data types.AgentData
	err := sonic.Unmarshal(ctx.Body(), &data)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	result, err := agentAction(uid, data)
	if err != nil {
		flog.Error(err)
		return protocol.ErrBadParam.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func (c *Controller) platformCallback(ctx *fiber.Ctx) error {
	platform := ctx.Params("platform")

	var err error
	switch platform {
	case tailchat.ID:
		err = tailchat.HandleHttp(ctx)
	case slack.ID:
		err = c.driver.HttpServer(ctx)
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

func (c *Controller) doWebhook(ctx *fiber.Ctx) error {
	flag := ctx.Params("flag")
	method := ctx.Method()

	flog.Info("[webhook] incoming %s flag: %s", method, flag)

	webhookRule, botHandler := bots.FindRuleAndHandler[webhook.Rule](flag, bots.List())

	if botHandler == nil {
		return protocol.ErrNotFound.New("bot not found")
	}

	typesCtx := types.Context{}

	var err error
	var find *model.Webhook
	if webhookRule.Secret {
		secret := ""
		val := ctx.FormValue("secret")
		if val != "" {
			secret = val
		}
		val = ctx.Query("secret")
		if val != "" {
			secret = val
		}
		val = ctx.Cookies("secret")
		if val != "" {
			secret = val
		}
		val = ctx.Get("X-Secret")
		if val != "" {
			secret = val
		}
		val = ctx.Get("Authorization")
		if val != "" {
			secret = strings.TrimPrefix(val, "Bearer ")
		}
		if secret == "" {
			return protocol.ErrParamVerificationFailed.New("secret not verification")
		}
		find, err = store.Database.GetWebhookBySecret(secret)
		if err != nil {
			return protocol.ErrNotAuthorized.Wrap(err)
		}
		if find.State != model.WebhookActive {
			return protocol.ErrAccessDenied.New("inactive")
		}

		typesCtx.AsUser = types.Uid(find.UID)
		typesCtx.Topic = find.Topic
	}

	var data []byte
	switch method {
	case http.MethodGet:
		data = ctx.Request().URI().QueryArgs().QueryString()
	case http.MethodPost:
		data = ctx.Body()
	}

	typesCtx.WebhookRuleId = flag
	typesCtx.Method = method
	typesCtx.Headers = ctx.GetReqHeaders()

	payload, err := botHandler.Webhook(typesCtx, data)
	if err != nil {
		return protocol.ErrFlagError.Wrap(err)
	}

	if find != nil {
		err = store.Database.IncreaseWebhookCount(find.ID)
		if err != nil {
			flog.Error(err)
		}
	}

	return ctx.JSON(payload)
}
