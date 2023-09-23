package markdown

import (
	"bytes"
	_ "embed"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"text/template"
)

const serviceVersion = "v1"

//go:embed markdown.html
var editorTemplate string

func editor(ctx *fiber.Ctx) error {
	flag := ctx.Params("flag")

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		return route.ErrorResponse(ctx, "flag error")
	}
	if p.IsExpired() {
		return route.ErrorResponse(ctx, "page expired")
	}

	t, err := template.New("tmpl").Parse(editorTemplate)
	if err != nil {
		return route.ErrorResponse(ctx, "page template error")
	}
	buf := bytes.NewBufferString("")
	p.Params["flag"] = flag
	data := p.Params
	err = t.Execute(buf, data)
	if err != nil {
		return route.ErrorResponse(ctx, "error execute")
	}

	return ctx.Send(buf.Bytes())
}

func saveMarkdown(ctx *fiber.Ctx) error {
	// data
	var data map[string]string
	err := ctx.BodyParser(&data)
	if err != nil {
		return route.ErrorResponse(ctx, "params error")
	}

	uid := data["uid"]
	flag := data["flag"]
	markdown := data["markdown"]
	if uid == "" || flag == "" || markdown == "" {
		return route.ErrorResponse(ctx, "params error")
	}

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		return route.ErrorResponse(ctx, "flag error")
	}
	if p.IsExpired() {
		return route.ErrorResponse(ctx, "page expired")
	}

	// store
	userUid := types.Uid(uid)
	botUid := types.Uid("") // fixme
	title := utils.MarkdownTitle(markdown)
	topic := "" // fixme
	payload := bots.StorePage(
		types.Context{AsUser: userUid, Original: botUid.String()},
		model.PageMarkdown, title,
		types.MarkdownMsg{Title: title, Raw: markdown})
	message := ""
	if link, ok := payload.(types.LinkMsg); ok {
		message = link.Url
	}

	// send
	err = event.Emit(event.SendEvent, types.KV{
		"topic":     topic,
		"topic_uid": botUid,
		"message":   message,
	})
	if err != nil {
		flog.Error(err)
		return ctx.SendString("send error")
	}

	return ctx.SendString("ok")
}
