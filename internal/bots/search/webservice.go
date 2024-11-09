package search

import (
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/internal/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/gofiber/fiber/v2"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/query", query),
	webservice.Get("/autocomplete", autocomplete),
}

// search everything
//
//	@Summary	search everything
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/search/query [get]
func query(ctx *fiber.Ctx) error {
	q := ctx.Query("q")
	source := ctx.Query("source")
	if q == "" {
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	list, _, err := search.NewClient().Search(source, q, 1, 10)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// search autocomplete
//
//	@Summary	search autocomplete
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/search/autocomplete [get]
func autocomplete(ctx *fiber.Ctx) error {
	q := ctx.Query("q")
	source := ctx.Query("source")
	if q == "" {
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	list, _, err := search.NewClient().Search(source, "title", 1, 10)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}
