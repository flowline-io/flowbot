package web

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var homeWebserviceRules = []webservice.Rule{
	webservice.Get("/home", homePage, route.WithNotAuth()),
}

func homePage(ctx fiber.Ctx) error {
	ctx.Type("html")
	return pages.HomePage().Render(context.Background(), ctx.Response().BodyWriter())
}
