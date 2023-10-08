package markdown

import (
	"bytes"
	_ "embed"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"text/template"
)

//go:embed markdown.html
var editorTemplate string

// markdown editor page
//
//	@Summary  markdown editor page
//	@Tags     markdown
//	@Produce  html
//	@Param    flag  path  string  true  "Flag"
//	@Router   /markdown/editor/{flag} [get]
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

// save markdown data
//
//	@Summary  save markdown data
//	@Tags     markdown
//	@Accept   json
//	@Produce  json
//	@Param    data  body      map[string]string  true  "Data"
//	@Success  200   {object}  protocol.Response
//	@Router   /markdown/data [post]
func saveMarkdown(ctx *fiber.Ctx) error {
	// data
	var data map[string]string
	err := ctx.BodyParser(&data)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	uid := data["uid"]
	flag := data["flag"]
	markdown := data["markdown"]
	if uid == "" || flag == "" || markdown == "" {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrFlagError))
	}
	if p.IsExpired() {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrFlagExpired))
	}

	// store
	userUid := types.Uid(uid)
	title := utils.MarkdownTitle(markdown)
	topic := "" // fixme
	payload := bots.StorePage(
		types.Context{AsUser: userUid, Original: Name},
		model.PageMarkdown, title,
		types.MarkdownMsg{Title: title, Raw: markdown})
	message := ""
	if link, ok := payload.(types.LinkMsg); ok {
		message = link.Url
	}

	// send
	err = event.Emit(event.SendEvent, types.KV{
		"topic":   topic,
		"bot":     Name,
		"message": message,
	})
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrSendMessageFailed))
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
