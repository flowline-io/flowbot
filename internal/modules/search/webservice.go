package search

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/query", query),
	webservice.Get("/autocomplete", autocomplete),
}

func query(ctx fiber.Ctx) error {
	q := ctx.Query("q")
	if q == "" {
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	if len(q) > validate.QueryMaxLen {
		return protocol.ErrBadParam.New("query too long")
	}

	source := ctx.Query("source")
	if len(source) > validate.NameMaxLen {
		return protocol.ErrBadParam.New("source too long")
	}

	list, _, err := search.Instance.Search(source, q, 1, 10)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}

func autocomplete(ctx fiber.Ctx) error {
	q := ctx.Query("q")
	if q == "" {
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	if len(q) > validate.QueryMaxLen {
		return protocol.ErrBadParam.New("query too long")
	}

	source := ctx.Query("source")
	if len(source) > validate.NameMaxLen {
		return protocol.ErrBadParam.New("source too long")
	}

	list, _, err := search.Instance.Search(source, "title", 1, 10)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewSuccessResponse(nil))
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}
