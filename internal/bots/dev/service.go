package dev

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/gofiber/fiber/v2"
)

const serviceVersion = "v1"

// example show example data
// @Summary      Show example
// @Description  get example data
// @Tags         dev
// @Accept       json
// @Produce      json
// @Success      200  {object}  types.ServerComMessage
// @Failure      400  {object}  types.ServerComMessage
// @Router       /bot/dev/v1/example [get]
func example(ctx *fiber.Ctx) error {
	return ctx.JSON(types.OkMessage(types.KV{
		"title": "example",
	}))
}
