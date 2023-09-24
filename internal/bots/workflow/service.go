package workflow

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/gofiber/fiber/v2"
)

const serviceVersion = "v1"

type rule struct {
	Bot          string            `json:"bot"`
	Id           string            `json:"id"`
	Title        string            `json:"title"`
	Desc         string            `json:"desc"`
	InputSchema  []types.FormField `json:"input_schema"`
	OutputSchema []types.FormField `json:"output_schema"`
}

// get chatbot actions
//
//	@Summary		get chatbot actions
//	@Description	get chatbot actions
//	@Tags			workflow
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	protocol.Response{data=map[string][]rule}
//	@Router			/workflow/v1/actions [get]
func actions(ctx *fiber.Ctx) error {
	result := make(map[string][]rule, len(bots.List()))
	for name, handler := range bots.List() {
		var list []rule
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []workflow.Rule:
				for _, item := range v {
					list = append(list, rule{
						Bot:          name,
						Id:           item.Id,
						Title:        item.Title,
						Desc:         item.Desc,
						InputSchema:  item.InputSchema,
						OutputSchema: item.OutputSchema,
					})
				}
			}
		}
		if len(list) > 0 {
			result[name] = list
		}
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}
