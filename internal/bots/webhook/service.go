package webhook

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/gofiber/fiber/v2"
	"io"
)

const serviceVersion = "v1"

func webhook(ctx *fiber.Ctx) error {
	flag := ctx.Params("flag")

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		return route.ErrorResponse(ctx, "flag error")
	}
	if p.IsExpired() {
		return route.ErrorResponse(ctx, "page expired")
	}

	//uid, _ := types.KV(p.Params).String("uid")
	//userUid := types.ParseUserId(uid)
	botUid := types.Uid(0) // fixme
	topic := ""            // fixme

	d, _ := io.ReadAll(ctx.Request().BodyStream())

	txt := ""
	if len(d) > 1000 {
		txt = fmt.Sprintf("[webhook:%s] body too long", flag)
	} else {
		txt = fmt.Sprintf("[webhook:%s] %s", flag, string(d))
	}
	// send
	err = event.Emit(event.SendEvent, types.KV{
		"topic":     topic,
		"topic_uid": botUid,
		"message":   txt,
	})
	if err != nil {
		flog.Error(err)
		return route.ErrorResponse(ctx, "error emit event")
	}

	return ctx.SendString("ok")
}
