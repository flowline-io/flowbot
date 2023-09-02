package server

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/page"
	"github.com/flowline-io/flowbot/internal/page/form"
	"github.com/flowline-io/flowbot/internal/page/library"
	"github.com/flowline-io/flowbot/internal/page/uikit"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/internal/platforms/tailchat"
	formRule "github.com/flowline-io/flowbot/internal/ruleset/form"
	pageRule "github.com/flowline-io/flowbot/internal/ruleset/page"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/queue"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	json "github.com/json-iterator/go"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
)

func setupMux(app *fiber.App) *http.ServeMux {
	// Webservice
	wc := route.NewContainer()
	for _, bot := range bots.List() {
		if ws := bot.Webservice(); ws != nil {
			wc.Add(ws)
		}
	}
	route.AddSwagger(wc)
	m := wc.ServeMux

	app.Group("/extra", adaptor.HTTPHandler(newRouter())) // todo remove extra prefix
	app.Group("/app", adaptor.HTTPHandler(newWebappRouter()))
	app.Group("/u", adaptor.HTTPHandler(newUrlRouter()))
	app.Group("/d", adaptor.HTTPHandler(newDownloadRouter()))

	app.All("/chatbot/:platform", platformCallback)

	return m
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	s := r.PathPrefix("/extra").Subrouter()
	s.Use(mux.CORSMethodMiddleware(r))
	// common
	s.HandleFunc("/oauth/{category}/{uid1}/{uid2}", storeOAuth)
	s.HandleFunc("/page/{id}", getPage)
	s.HandleFunc("/form", postForm).Methods(http.MethodPost)
	s.HandleFunc("/queue/stats", queueStats)
	s.HandleFunc("/p/{id}/{flag}", renderPage)
	// bot
	s.HandleFunc("/linkit", linkitData)
	s.HandleFunc("/session", wbSession)

	return s
}

func newWebappRouter() *mux.Router {
	r := mux.NewRouter()
	s := r.PathPrefix("/app").Subrouter()
	for name, bot := range bots.List() {
		if f := bot.Webapp(); f != nil {
			s.HandleFunc(fmt.Sprintf("/%s/{subpath:.*}", name), f)
		}
	}
	return s
}

func newUrlRouter() *mux.Router {
	r := mux.NewRouter()
	s := r.PathPrefix("/u").Subrouter()
	s.HandleFunc("/{flag}", urlRedirect)
	return s
}

func newDownloadRouter() *mux.Router {
	dir := config.App.DownloadPath
	r := mux.NewRouter()
	r.PathPrefix("/d").Handler(http.StripPrefix("/d/", http.FileServer(http.Dir(dir))))
	return r
}

// handler

func storeOAuth(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["provider"]
	flag := vars["flag"]

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		errorResponse(rw, "flag error")
		return
	}
	if p.IsExpired() {
		errorResponse(rw, "oauth expired")
		return
	}

	uid, _ := types.KV(p.Params).String("uid")
	topic, _ := types.KV(p.Params).String("topic")
	if uid == "" || topic == "" {
		errorResponse(rw, "path error")
		return
	}

	// code -> token
	provider := newProvider(name)
	tk, err := provider.GetAccessToken(req)
	if err != nil {
		flog.Error(err)
		errorResponse(rw, "oauth error")
		return
	}

	// store
	extra := types.KV{}
	_ = extra.Scan(tk["extra"])
	err = store.Chatbot.OAuthSet(model.OAuth{
		UID:   uid,
		Topic: topic,
		Name:  name,
		Type:  name,
		Token: tk["token"].(string),
		Extra: model.JSON(extra),
	})
	if err != nil {
		flog.Error(err)
		errorResponse(rw, "store error")
		return
	}

	_, _ = rw.Write([]byte("ok"))
}

func getPage(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	p, err := store.Chatbot.PageGet(id)
	if err != nil {
		flog.Error(err)
		errorResponse(rw, "page not found")
		return
	}

	title, _ := types.KV(p.Schema).String("title")
	if title == "" {
		title = "Page"
	}
	var comp app.UI
	switch p.Type {
	case model.PageForm:
		f, _ := store.Chatbot.FormGet(p.PageID)
		comp = page.RenderForm(p, f)
	case model.PageTable:
		comp = page.RenderTable(p)
	case model.PageShare:
		comp = page.RenderShare(p)
	case model.PageJson:
		comp = page.RenderJson(p)
	case model.PageHtml:
		comp = page.RenderHtml(p)
	case model.PageMarkdown:
		comp = page.RenderMarkdown(p)
	case model.PageChart:
		d, err := json.Marshal(p.Schema)
		if err != nil {
			return
		}
		var msg types.ChartMsg
		err = json.Unmarshal(d, &msg)
		if err != nil {
			return
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

		_ = line.Render(rw)
		return
	default:
		errorResponse(rw, "page error type")
		return
	}

	_, _ = rw.Write([]byte(fmt.Sprintf(page.Layout, title,
		library.UIKitCss, library.UIKitJs, library.UIKitIconJs,
		"", app.HTMLString(uikit.Container(comp)), "")))
}

func renderPage(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	pageRuleId := vars["id"]
	flag := vars["flag"]

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		errorResponse(rw, "flag error")
		return
	}
	if p.IsExpired() {
		errorResponse(rw, "page expired")
		return
	}

	kv := types.KV(p.Params)
	original, _ := kv.String("original")
	topic, _ := kv.String("topic")
	uid, _ := kv.String("uid")

	ctx := types.Context{
		Original:   original,
		RcptTo:     topic,
		AsUser:     types.ParseUserId(uid),
		PageRuleId: pageRuleId,
	}

	var botHandler bots.Handler
	for _, handler := range bots.List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []pageRule.Rule:
				for _, rule := range v {
					if rule.Id == pageRuleId {
						botHandler = handler
					}
				}
			}
		}
	}

	if botHandler == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	html, err := botHandler.Page(ctx, flag)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write([]byte("404 not found"))
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
		_, _ = rw.Write([]byte("error page"))
		return
	}
	_, _ = rw.Write([]byte(html))
}

func postForm(rw http.ResponseWriter, req *http.Request) {
	_ = req.ParseForm()
	pf := req.PostForm

	formId := pf.Get("x-form_id")
	uid := pf.Get("x-uid")
	//uid2 := pf.Get("x-topic")

	//userUid := types.ParseUserId(uid)
	//topicUid := types.ParseUserId(uid2)
	topic := "" // fixme

	formData, err := store.Chatbot.FormGet(formId)
	if err != nil {
		return
	}
	if formData.State == model.FormStateSubmitSuccess || formData.State == model.FormStateSubmitFailed {
		return
	}

	values := make(map[string]interface{})
	d, err := json.Marshal(formData.Schema)
	if err != nil {
		return
	}
	var formMsg types.FormMsg
	err = json.Unmarshal(d, &formMsg)
	if err != nil {
		return
	}
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
			value := pf.Get(field.Key)
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
		_, _ = rw.Write([]byte(err.Error()))
		return
	}

	ctx := types.Context{
		//Original:   topicUid.UserId(),
		RcptTo:     topic,
		AsUser:     types.ParseUserId(uid),
		FormId:     formData.FormID,
		FormRuleId: formMsg.ID,
	}

	// user auth record

	// get bot handler
	formRuleId, ok := types.KV(formData.Schema).String("id")
	if !ok {
		flog.Error(fmt.Errorf("form %s %s", formId, "error form rule id"))
		return
	}
	var botHandler bots.Handler
	for _, handler := range bots.List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []formRule.Rule:
				for _, rule := range v {
					if rule.Id == formRuleId {
						botHandler = handler
					}
				}
			}
		}
	}

	if botHandler != nil {
		if !botHandler.IsReady() {
			flog.Info("bot %s unavailable", topic)
			return
		}

		// form message
		payload, err := botHandler.Form(ctx, values)
		if err != nil {
			flog.Warn("topic[%s]: failed to form bot: %v", topic, err)
			return
		}

		// stats
		stats.Inc("BotRunFormTotal", 1)

		// send message
		if payload == nil {
			return
		}

		topicUid := types.Uid(0) // fixme
		botSend(topic, topicUid, payload)

		// pipeline form stage
		pipelineFlag, _ := types.KV(formData.Extra).String("pipeline_flag")
		pipelineVersion, _ := types.KV(formData.Extra).Int64("pipeline_version")
		nextPipeline(ctx, pipelineFlag, int(pipelineVersion), topic, topicUid)
	}

	_, _ = rw.Write([]byte("ok"))
}

func linkitData(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "*")
	if req.Method == http.MethodOptions {
		return
	}

	// authorization
	uid, isValid := route.CheckAccessToken(route.GetAccessToken(req))
	if !isValid {
		errorResponse(rw, "401")
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		errorResponse(rw, "error")
		return
	}

	var data types.LinkData
	err = json.Unmarshal(body, &data)
	if err != nil {
		errorResponse(rw, "error")
		return
	}

	result, err := linkitAction(uid, data)
	if err != nil {
		errorResponse(rw, "error")
		return
	}
	fmt.Println(result)
	//res, _ := json.Marshal(types.ServerComMessage{
	//	Code: http.StatusOK,
	//	Data: result,
	//})
	//rw.Header().Set("Content-Type", "application/json")
	//_, _ = rw.Write(res)
}

func urlRedirect(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	flag, ok := vars["flag"]
	if !ok {
		errorResponse(rw, "error")
		return
	}

	url, err := store.Chatbot.UrlGetByFlag(flag)
	if err != nil {
		errorResponse(rw, "error")
		return
	}

	// view count
	_ = store.Chatbot.UrlViewIncrease(flag)

	// redirect
	http.Redirect(rw, req, url.URL, http.StatusFound)
}

func queueStats(rw http.ResponseWriter, _ *http.Request) {
	html, err := queue.Stats()
	if err != nil {
		errorResponse(rw, "queue stats error")
		return
	}
	_, _ = fmt.Fprint(rw, html)
}

func wbSession(wrt http.ResponseWriter, req *http.Request) {
	uid, isValid := route.CheckAccessToken(route.GetAccessToken(req))
	if !isValid {
		wrt.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(wrt).Encode(types.ErrMessage(http.StatusForbidden, "Missing, invalid or expired access token"))
		return
	}

	if req.Method != http.MethodGet {
		wrt.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(wrt).Encode(types.ErrMessage(http.StatusBadRequest, "invalid http method"))
		return
	}

	ws, err := upgrader.Upgrade(wrt, req, nil)
	if errors.As(err, &websocket.HandshakeError{}) {
		flog.Error(err)
		return
	} else if err != nil {
		flog.Error(err)
		return
	}

	sess, count := sessionStore.NewSession(ws, "")
	if globals.useXForwardedFor {
		sess.remoteAddr = req.Header.Get("X-Forwarded-For")
		if !utils.IsRoutableIP(sess.remoteAddr) {
			sess.remoteAddr = ""
		}
	}
	if sess.remoteAddr == "" {
		sess.remoteAddr = req.RemoteAddr
	}
	sess.uid = uid

	flog.Info("linkit: session started %v %v %v", sess.sid, sess.remoteAddr, count)

	// Do work in goroutines to return from serveWebSocket() to release file pointers.
	// Otherwise, "too many open files" will happen.
	go sess.writeLoop()
	go sess.readLoop()
}

func platformCallback(ctx *fiber.Ctx) error {
	platform := ctx.Params("platform")

	var err error
	switch platform {
	case tailchat.ID:
		err = tailchat.HandleHttp(ctx)
	case slack.ID:
		d := slack.Driver{}
		err = d.HttpServer(ctx)
	}
	if err != nil {
		flog.Error(err)
		var protocolError *protocol.Error
		if errors.As(err, protocolError) {
			return ctx.JSON(protocol.NewFailedResponse(protocolError))
		}
		return err
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
