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

// workflow list
//
//	@Summary  workflow list
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Success  200  {object}  protocol.Response{data=[]model.Workflow}
//	@Router   /workflow/workflows [get]
func workflowList(ctx *fiber.Ctx) error {
	return nil
}

// workflow detail
//
//	@Summary  workflow detail
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "ID"
//	@Success  200  {object}  protocol.Response{data=model.Workflow}
//	@Router   /workflow/workflow/{id} [get]
func workflowDetail(ctx *fiber.Ctx) error {
	return nil
}

// workflow create
//
//	@Summary  workflow create
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    workflow  body      model.Workflow  true  "workflow data"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/workflow [post]
func workflowCreate(ctx *fiber.Ctx) error {
	return nil
}

// workflow update
//
//	@Summary  workflow update
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "ID"
//	@Param    workflow  body      model.Workflow  true  "workflow data"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/workflow/{id} [put]
func workflowUpdate(ctx *fiber.Ctx) error {
	return nil
}

// workflow delete
//
//	@Summary  workflow delete
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "ID"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/workflow/{id} [delete]
func workflowDelete(ctx *fiber.Ctx) error {
	return nil
}

// workflow trigger list
//
//	@Summary  workflow trigger list
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Workflow ID"
//	@Success  200  {object}  protocol.Response{data=[]model.WorkflowTrigger}
//	@Router   /workflow/workflow/{id}/triggers [get]
func workflowTriggerList(ctx *fiber.Ctx) error {
	return nil
}

// workflow trigger create
//
//	@Summary  workflow trigger create
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Workflow ID"
//	@Param    trigger  body      model.WorkflowTrigger  true  "workflow trigger data"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/workflow/{id}/trigger [post]
func workflowTriggerCreate(ctx *fiber.Ctx) error {
	return nil
}

// workflow trigger update
//
//	@Summary  workflow trigger update
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Trigger ID"
//	@Param    trigger  body      model.WorkflowTrigger  true  "workflow trigger data"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/trigger/{id} [put]
func workflowTriggerUpdate(ctx *fiber.Ctx) error {
	return nil
}

// workflow trigger delete
//
//	@Summary  workflow trigger delete
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Trigger ID"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/trigger/{id} [delete]
func workflowTriggerDelete(ctx *fiber.Ctx) error {
	return nil
}

// workflow job list
//
//	@Summary  workflow job list
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Workflow ID"
//	@Success  200  {object}  protocol.Response{data=[]model.Job}
//	@Router   /workflow/workflow/{id}/jobs [get]
func workflowJobList(ctx *fiber.Ctx) error {
	return nil
}

// workflow job detail
//
//	@Summary  workflow job detail
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Job ID"
//	@Success  200  {object}  protocol.Response{data=model.Job}
//	@Router   /workflow/job/{id} [get]
func workflowJobDetail(ctx *fiber.Ctx) error {
	return nil
}

// workflow job rerun
//
//	@Summary  workflow job rerun
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Job ID"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/job/{id}/rerun [post]
func workflowJobRerun(ctx *fiber.Ctx) error {
	return nil
}

// workflow dag detail
//
//	@Summary  workflow dag detail
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Workflow ID"
//	@Success  200  {object}  protocol.Response{data=model.Dag}
//	@Router   /workflow/workflow/{id}/dag [get]
func workflowDagDetail(ctx *fiber.Ctx) error {
	return nil
}

// workflow dag update
//
//	@Summary  workflow dag update
//	@Tags     workflow
//	@Accept   json
//	@Produce  json
//	@Param    id  path      int  true  "Workflow ID"
//	@Param    trigger  body      model.Dag  true  "workflow dag data"
//	@Success  200  {object}  protocol.Response
//	@Router   /workflow/workflow/{id}/dag [put]
func workflowDagUpdate(ctx *fiber.Ctx) error {
	return nil
}
