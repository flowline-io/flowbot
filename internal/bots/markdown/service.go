package markdown

import (
	"bytes"
	_ "embed"
	"github.com/emicklei/go-restful/v3"
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/event"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/route"
	"github.com/sysatom/flowbot/pkg/utils"
	"text/template"
)

const serviceVersion = "v1"

//go:embed markdown.html
var editorTemplate string

func editor(req *restful.Request, resp *restful.Response) {
	flag := req.PathParameter("flag")

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		route.ErrorResponse(resp, "flag error")
		return
	}
	if p.IsExpired() {
		route.ErrorResponse(resp, "page expired")
		return
	}

	t, err := template.New("tmpl").Parse(editorTemplate)
	if err != nil {
		route.ErrorResponse(resp, "page template error")
		return
	}
	buf := bytes.NewBufferString("")
	p.Params["flag"] = flag
	data := p.Params
	err = t.Execute(buf, data)
	if err != nil {
		route.ErrorResponse(resp, "error execute")
		return
	}

	_, _ = resp.Write(buf.Bytes())
}

func saveMarkdown(req *restful.Request, resp *restful.Response) {
	// data
	var data map[string]string
	err := req.ReadEntity(&data)
	if err != nil {
		route.ErrorResponse(resp, "params error")
		return
	}

	uid := data["uid"]
	flag := data["flag"]
	markdown := data["markdown"]
	if uid == "" || flag == "" || markdown == "" {
		route.ErrorResponse(resp, "params error")
		return
	}

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		route.ErrorResponse(resp, "flag error")
		return
	}
	if p.IsExpired() {
		route.ErrorResponse(resp, "page expired")
		return
	}

	// store
	userUid := types.ParseUserId(uid)
	botUid := types.Uid(0) // fixme
	title := utils.MarkdownTitle(markdown)
	topic := "" // fixme
	payload := bots.StorePage(
		types.Context{AsUser: userUid, Original: botUid.UserId()},
		model.PageMarkdown, title,
		types.MarkdownMsg{Title: title, Raw: markdown})
	message := ""
	if link, ok := payload.(types.LinkMsg); ok {
		message = link.Url
	}

	// send
	err = event.Emit(event.SendEvent, types.KV{
		"topic":     topic,
		"topic_uid": int64(botUid),
		"message":   message,
	})
	if err != nil {
		logs.Err.Println(err)
		_, _ = resp.Write([]byte("send error"))
		return
	}

	_, _ = resp.Write([]byte("ok"))
}
