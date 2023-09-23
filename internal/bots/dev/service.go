package dev

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/gofiber/fiber/v2"
)

const serviceVersion = "v1"

func example(ctx *fiber.Ctx) error {
	return ctx.JSON(types.KV{
		"title": "example",
	})
}
