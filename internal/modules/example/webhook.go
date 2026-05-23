package example

import (
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webhookRules = []webservice.Rule{
	webservice.Post("/webhook/example", exampleWebhook, route.WithNotAuth()),
}

// exampleWebhook handles POST /service/example/webhook/example
//
//	@Summary	Receive example webhook events
//	@Tags		example
//	@Accept		json
//	@Produce	json
//	@Success	202	{string}	string	"Accepted"
//	@Router		/example/webhook/example [post]
func exampleWebhook(ctx fiber.Ctx) error {
	return ctx.SendStatus(fiber.StatusAccepted)
}
