package user

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/internal/types/ruleset/webservice"
	"github.com/gofiber/fiber/v2"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/dashboard", dashboard),
}

// dashboard show dashboard data
//
//	@Summary	Show dashboard
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/user/dashboard [get]
func dashboard(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"title": "example",
	}))
}
