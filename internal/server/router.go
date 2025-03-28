package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	jsoniter "github.com/json-iterator/go"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"gorm.io/gorm"
)

func setupMux(a *fiber.App) {
	// Webservice
	for _, bot := range bots.List() {
		bot.Webservice(a)
	}

	// common
	a.Get("/", func(c *fiber.Ctx) error { return nil })
	a.All("/oauth/:provider/:flag", storeOAuth)
	a.Get("/p/:id", getPage)
	// form
	a.Post("/form", postForm)
	// page
	a.Get("/page/:id/:flag", renderPage)
	// agent
	a.Post("/agent", agentData)
	// webhook
	a.All("/webhook/:flag", doWebhook)
	// platform
	a.All("/chatbot/:platform", platformCallback)
}

// handler

func storeOAuth(ctx *fiber.Ctx) error {
	name := ctx.Params("provider")
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(flag)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrFlagError, err))
	}
	if p.IsExpired() {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrFlagExpired))
	}

	uid, _ := types.KV(p.Params).String("uid")
	topic, _ := types.KV(p.Params).String("topic")
	if uid == "" {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	// code -> token
	provider := newProvider(name)
	tk, err := provider.GetAccessToken(ctx)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrOAuthError, err))
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrOAuthError, err))
	}

	return ctx.SendString("ok")
}

func getPage(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	p, err := store.Database.PageGet(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrNotFound, err))
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
		d, err := jsoniter.Marshal(p.Schema)
		if err != nil {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
		}
		var msg types.ChartMsg
		err = jsoniter.Unmarshal(d, &msg)
		if err != nil {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
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
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadRequest))
	}

	return ctx.SendString(page.RenderComponent(title, comp))
}

func renderPage(ctx *fiber.Ctx) error {
	pageRuleId := ctx.Params("id")
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(flag)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrFlagError, err))
	}
	if p.IsExpired() {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrFlagExpired))
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
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrNotFound))
	}

	html, err := botHandler.Page(typesCtx, flag, args)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrNotFound, err))
		}
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
	}
	ctx.Set("Content-Type", "text/html")
	return ctx.SendString(html)
}

func postForm(ctx *fiber.Ctx) error {
	formId := ctx.FormValue("x-form_id")
	uid := ctx.FormValue("x-uid")
	topic := ctx.FormValue("x-topic")

	formData, err := store.Database.FormGet(formId)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
	}
	if formData.State == model.FormStateSubmitSuccess || formData.State == model.FormStateSubmitFailed {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, errors.New("error form state")))
	}

	values := make(types.KV)
	d, err := jsoniter.Marshal(formData.Schema)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
	}
	var formMsg types.FormMsg
	err = jsoniter.Unmarshal(d, &formMsg)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
	}
	f, err := ctx.MultipartForm()
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, fmt.Errorf("form %s %s", formId, "error form rule id")))
	}

	_, botHandler := bots.FindRuleAndHandler[formRule.Rule](formRuleId, bots.List())

	if botHandler != nil {
		if !botHandler.IsReady() {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, fmt.Errorf("bot %s unavailable", topic)))
		}

		// form message
		payload, err := botHandler.Form(botCtx, values)
		if err != nil {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
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

func agentData(ctx *fiber.Ctx) error {
	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(ctx.Context(), &r, true); err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(protocol.NewFailedResponseWithError(protocol.ErrInternalServerError, err))
	}
	// authorization
	uid, isValid := route.CheckAccessToken(route.GetAccessToken(&r))
	if !isValid {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized))
	}

	var data types.AgentData
	err := jsoniter.Unmarshal(ctx.Body(), &data)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
	}

	result, err := agentAction(uid, data)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func platformCallback(ctx *fiber.Ctx) error {
	platform := ctx.Params("platform")

	var err error
	switch platform {
	case tailchat.ID:
		err = tailchat.HandleHttp(ctx)
	case slack.ID:
		err = slack.NewDriver().HttpServer(ctx)
	}
	if err != nil {
		var protocolError *protocol.Error
		if errors.As(err, protocolError) {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocolError, err))
		}
		return err
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

func doWebhook(ctx *fiber.Ctx) error {
	flag := ctx.Params("flag")
	method := ctx.Method()

	flog.Info("[webhook] incoming %s flag: %s", method, flag)

	webhookRule, botHandler := bots.FindRuleAndHandler[webhook.Rule](flag, bots.List())

	if botHandler == nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrNotFound))
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
			return ctx.JSON(protocol.NewFailedResponse(protocol.ErrParamVerificationFailed))
		}
		find, err = store.Database.GetWebhookBySecret(secret)
		if err != nil {
			return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrNotAuthorized, err))
		}
		if find.State != model.WebhookActive {
			return ctx.JSON(protocol.NewFailedResponse(protocol.ErrAccessDenied))
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrFlagError, err))
	}

	if find != nil {
		err = store.Database.IncreaseWebhookCount(find.ID)
		if err != nil {
			flog.Error(err)
		}
	}

	return ctx.JSON(payload)
}
