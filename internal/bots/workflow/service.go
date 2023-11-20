package workflow

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/gofiber/fiber/v2"
)

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
//	@Summary  get chatbot actions
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Success  200  {object}  protocol.Response{data=map[string][]rule}
//	@Router   /workflow/actions [get]
func actions(ctx *fiber.Ctx) error {
	result := make(map[string][]rule, len(bots.List()))
	for name, botHandler := range bots.List() {
		var list []rule
		for _, item := range botHandler.Rules() {
			switch v := item.(type) {
			case []workflow.Rule:
				for _, ruleItem := range v {
					list = append(list, rule{
						Bot:          name,
						Id:           ruleItem.Id,
						Title:        ruleItem.Title,
						Desc:         ruleItem.Desc,
						InputSchema:  ruleItem.InputSchema,
						OutputSchema: ruleItem.OutputSchema,
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

func example(ctx *fiber.Ctx) error {
	return ctx.SendString("example")

}
