package search

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/query", query),
	webservice.Get("/autocomplete", autocomplete),
}

// search everything
//
//	@Summary	search everything
//	@Tags		search
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/search/query [get]
func query(ctx fiber.Ctx) error {
	q := ctx.Query("q")
	source := ctx.Query("source")
	if q == "" {
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	list, _, err := search.Instance.Search(source, q, 1, 10)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// search autocomplete
//
//	@Summary	search autocomplete
//	@Tags		search
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/search/autocomplete [get]
func autocomplete(ctx fiber.Ctx) error {
	q := ctx.Query("q")
	source := ctx.Query("source")
	if q == "" {
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	list, _, err := search.Instance.Search(source, "title", 1, 10)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}
