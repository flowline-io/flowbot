package server

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/sysatom/flowbot/internal/bots"
	compPage "github.com/sysatom/flowbot/internal/page"
	form2 "github.com/sysatom/flowbot/internal/page/form"
	"github.com/sysatom/flowbot/internal/page/library"
	"github.com/sysatom/flowbot/internal/page/uikit"
	"github.com/sysatom/flowbot/internal/ruleset/form"
	"github.com/sysatom/flowbot/internal/ruleset/page"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/queue"
	"github.com/sysatom/flowbot/pkg/route"
	"github.com/sysatom/flowbot/pkg/utils"
	"gorm.io/gorm"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

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
	dir := os.Getenv("DOWNLOAD_PATH")
	r := mux.NewRouter()
	r.PathPrefix("/d").Handler(http.StripPrefix("/d/", http.FileServer(http.Dir(dir))))
	return r
}

// handler

func storeOAuth(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	category := vars["category"]
	ui1, _ := strconv.ParseUint(vars["uid1"], 10, 64)
	ui2, _ := strconv.ParseUint(vars["uid2"], 10, 64)
	if ui1 == 0 || ui2 == 0 {
		errorResponse(rw, "path error")
		return
	}

	// code -> token
	provider := newProvider(category)
	tk, err := provider.StoreAccessToken(req)
	if err != nil {
		logs.Err.Println("router oauth", err)
		errorResponse(rw, "oauth error")
		return
	}

	// store
	extra := types.KV{}
	_ = extra.Scan(tk["extra"])
	err = store.Chatbot.OAuthSet(model.OAuth{
		UID:   types.Uid(ui1).UserId(),
		Topic: types.Uid(ui2).UserId(),
		Name:  category,
		Type:  category,
		Token: tk["token"].(string),
		Extra: model.JSON(extra),
	})
	if err != nil {
		logs.Err.Println("router oauth", err)
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
		logs.Err.Println(err)
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
		comp = compPage.RenderForm(p, f)
	case model.PageTable:
		comp = compPage.RenderTable(p)
	case model.PageShare:
		comp = compPage.RenderShare(p)
	case model.PageJson:
		comp = compPage.RenderJson(p)
	case model.PageHtml:
		comp = compPage.RenderHtml(p)
	case model.PageMarkdown:
		comp = compPage.RenderMarkdown(p)
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

	_, _ = rw.Write([]byte(fmt.Sprintf(compPage.Layout, title,
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
			case []page.Rule:
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

	formBuilder := form2.NewBuilder(formMsg.Field)
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
		logs.Err.Printf("form %s %s", formId, "error form rule id")
		return
	}
	var botHandler bots.Handler
	for _, handler := range bots.List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []form.Rule:
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
			logs.Info.Printf("bot %s unavailable", topic)
			return
		}

		// form message
		payload, err := botHandler.Form(ctx, values)
		if err != nil {
			logs.Warn.Printf("topic[%s]: failed to form bot: %v", topic, err)
			return
		}

		// stats
		statsInc("BotRunFormTotal", 1)

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
	res, _ := json.Marshal(types.ServerComMessage{
		Code: http.StatusOK,
		Data: result,
	})
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(res)
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
	return
}

func queueStats(rw http.ResponseWriter, _ *http.Request) {
	html, err := queue.Stats()
	if err != nil {
		errorResponse(rw, "queue stats error")
		return
	}
	_, _ = fmt.Fprint(rw, html)
}

// globals session
var sessionStore = NewExtraSessionStore(idleSessionTimeout + 15*time.Second)

func wbSession(wrt http.ResponseWriter, req *http.Request) {
	uid, isValid := route.CheckAccessToken(route.GetAccessToken(req))
	if !isValid {
		wrt.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(wrt).Encode(types.ErrMessage(http.StatusForbidden, "Missing, invalid or expired access token"))
		logs.Err.Println("ws: Missing, invalid or expired API key")
		return
	}

	if req.Method != http.MethodGet {
		wrt.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(wrt).Encode(types.ErrMessage(http.StatusBadRequest, "invalid http method"))
		logs.Err.Println("ws: Invalid HTTP method", req.Method)
		return
	}

	ws, err := upgrader.Upgrade(wrt, req, nil)
	if errors.As(err, &websocket.HandshakeError{}) {
		logs.Err.Println("ws: Not a websocket handshake")
		return
	} else if err != nil {
		logs.Err.Println("ws: failed to Upgrade ", err)
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

	logs.Info.Println("linkit: session started", sess.sid, sess.remoteAddr, count)

	// Do work in goroutines to return from serveWebSocket() to release file pointers.
	// Otherwise, "too many open files" will happen.
	go sess.writeLoop()
	go sess.readLoopExtra()
}
