package webhook

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/gofiber/fiber/v2"
	"io"
)

// trigger webhook
//
//	@Summary  trigger webhook
//	@Tags     webhook
//	@Accept   json
//	@Produce  json
//	@Param    flag  path      string  true  "Flag"
//	@Success  200   {object}  protocol.Response
//	@Router   /webhook/trigger/{flag} [post]
func webhook(ctx *fiber.Ctx) error {
	flag := ctx.Params("flag")

	p, err := store.Database.ParameterGet(flag)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrFlagError, err))
	}
	if p.IsExpired() {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrFlagExpired))
	}

	//uid, _ := types.KV(p.Params).String("uid")
	//userUid := types.ParseUserId(uid)
	topic := "" // fixme

	d, _ := io.ReadAll(ctx.Request().BodyStream())

	txt := ""
	if len(d) > 1000 {
		txt = fmt.Sprintf("[webhook:%s] body too long", flag)
	} else {
		txt = fmt.Sprintf("[webhook:%s] %s", flag, string(d))
	}
	// send
	err = event.PublishMessage(protocol.MessageDirectEvent, types.KV{
		"topic":   topic,
		"bot":     Name,
		"message": txt,
	})
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrEmitEventError, err))
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
