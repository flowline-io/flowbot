package dev

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/gofiber/fiber/v2"
)

const serviceVersion = "v1"

// example show example data
//
//	@Summary		Show example
//	@Description	get example data
//	@Tags			dev
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	protocol.Response
//	@Router			/dev/v1/example [get]
func example(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"title": "example",
	}))
}
