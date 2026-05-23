package example

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/ability"
	abilityexample "github.com/flowline-io/flowbot/pkg/ability/example"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/example", example, route.WithNotAuth()),
	webservice.Get("/get", getExampleItem, route.WithNotAuth()),
	webservice.Get("/health", healthExample, route.WithNotAuth()),
	webservice.Post("/create", createExampleItem, route.WithNotAuth()),
	webservice.Delete("/delete", deleteExampleItem, route.WithNotAuth()),
}

// example show example data
//
//	@Summary	Show example
//	@Tags		example
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/example/example [get]
func example(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"title": "example",
		"cpu":   "20%",
		"mem":   "50%",
		"disk":  "70%",
	}))
}

// getExampleItem handles GET /service/example/get?id=xxx
//
//	@Summary	Get example item
//	@Tags		example
//	@Param		id	query	string	true	"item id"
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/get [get]
func getExampleItem(ctx fiber.Ctx) error {
	id := ctx.Query("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	res, err := ability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleGet, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// healthExample handles GET /service/example/health
//
//	@Summary	Example health check
//	@Tags		example
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/health [get]
func healthExample(ctx fiber.Ctx) error {
	res, err := ability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleHealth, map[string]any{})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// createExampleItem handles POST /service/example/create
//
//	@Summary	Create example item
//	@Tags		example
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/create [post]
func createExampleItem(ctx fiber.Ctx) error {
	var body struct {
		Title string `json:"title"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	if body.Title == "" {
		return types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	res, err := ability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleCreate, map[string]any{"title": body.Title})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// deleteExampleItem handles DELETE /service/example/delete?id=xxx
//
//	@Summary	Delete example item
//	@Tags		example
//	@Param		id	query	string	true	"item id"
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/delete [delete]
func deleteExampleItem(ctx fiber.Ctx) error {
	id := ctx.Query("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	res, err := ability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleDelete, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}
