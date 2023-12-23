package workflow

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/route"
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
//	@Summary	get chatbot actions
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=map[string][]rule}
//	@Router		/workflow/actions [get]
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

// workflow list
//
//	@Summary	workflow list
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=[]model.Workflow}
//	@Router		/workflow/workflows [get]
func workflowList(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)

	list, err := store.Database.ListWorkflows(uid, topic)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// workflow detail
//
//	@Summary	workflow detail
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"ID"
//	@Success	200	{object}	protocol.Response{data=model.Workflow}
//	@Router		/workflow/workflow/{id} [get]
func workflowDetail(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	item, err := store.Database.GetWorkflow(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(item))
}

// workflow create
//
//	@Summary	workflow create
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		script	body		model.WorkflowScript	true	"workflow script data"
//	@Success	200			{object}	protocol.Response
//	@Router		/workflow/workflow [post]
func workflowCreate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)

	script := new(model.WorkflowScript)
	err := ctx.BodyParser(&script)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	if script.Lang != model.WorkflowScriptYaml {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrUnsupported))
	}

	wf, triggers, dag, err := ParseYamlWorkflow(script.Code)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	wf.UID = uid.String()
	wf.Topic = topic
	_, err = store.Database.CreateWorkflow(wf, script, dag, triggers)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// workflow update
//
//	@Summary	workflow update
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id			path		int				true	"ID"
//	@Param		script	body		model.WorkflowScript	true	"workflow script data"
//	@Success	200			{object}	protocol.Response
//	@Router		/workflow/workflow/{id} [put]
func workflowUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	id := route.GetIntParam(ctx, "id")

	script := new(model.WorkflowScript)
	err := ctx.BodyParser(&script)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	if script.Lang != model.WorkflowScriptYaml {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrUnsupported))
	}

	// todo script update

	item := new(model.Workflow)
	err = ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}
	item.UID = uid.String()
	item.Topic = topic
	item.ID = id
	err = store.Database.UpdateWorkflow(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// workflow delete
//
//	@Summary	workflow delete
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"ID"
//	@Success	200	{object}	protocol.Response
//	@Router		/workflow/workflow/{id} [delete]
func workflowDelete(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	err := store.Database.DeleteWorkflow(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// workflow trigger list
//
//	@Summary	workflow trigger list
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"Workflow ID"
//	@Success	200	{object}	protocol.Response{data=[]model.WorkflowTrigger}
//	@Router		/workflow/workflow/{id}/triggers [get]
func workflowTriggerList(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	item, err := store.Database.GetWorkflow(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(item.Triggers))
}

// workflow trigger create
//
//	@Summary	workflow trigger create
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int						true	"Workflow ID"
//	@Param		trigger	body		model.WorkflowTrigger	true	"workflow trigger data"
//	@Success	200		{object}	protocol.Response
//	@Router		/workflow/workflow/{id}/trigger [post]
func workflowTriggerCreate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	id := route.GetIntParam(ctx, "id")

	item := new(model.WorkflowTrigger)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	item.UID = uid.String()
	item.Topic = topic
	item.WorkflowID = id
	_, err = store.Database.CreateWorkflowTrigger(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// workflow trigger update
//
//	@Summary	workflow trigger update
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int						true	"Trigger ID"
//	@Param		trigger	body		model.WorkflowTrigger	true	"workflow trigger data"
//	@Success	200		{object}	protocol.Response
//	@Router		/workflow/trigger/{id} [put]
func workflowTriggerUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	id := route.GetIntParam(ctx, "id")

	item := new(model.WorkflowTrigger)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	item.UID = uid.String()
	item.Topic = topic
	item.ID = id
	err = store.Database.UpdateWorkflowTrigger(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// workflow trigger delete
//
//	@Summary	workflow trigger delete
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"Trigger ID"
//	@Success	200	{object}	protocol.Response
//	@Router		/workflow/trigger/{id} [delete]
func workflowTriggerDelete(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	err := store.Database.DeleteWorkflowTrigger(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// workflow job list
//
//	@Summary	workflow job list
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"Workflow ID"
//	@Success	200	{object}	protocol.Response{data=[]model.Job}
//	@Router		/workflow/workflow/{id}/jobs [get]
func workflowJobList(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	list, err := store.Database.GetJobsByWorkflowId(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// workflow job detail
//
//	@Summary	workflow job detail
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"Job ID"
//	@Success	200	{object}	protocol.Response{data=model.Job}
//	@Router		/workflow/job/{id} [get]
func workflowJobDetail(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	item, err := store.Database.GetJob(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(item))
}

// workflow job rerun
//
//	@Summary	workflow job rerun
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"Job ID"
//	@Success	200	{object}	protocol.Response
//	@Router		/workflow/job/{id}/rerun [post]
func workflowJobRerun(ctx *fiber.Ctx) error {
	return nil
}

// workflow dag detail
//
//	@Summary	workflow dag detail
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"Workflow ID"
//	@Success	200	{object}	protocol.Response{data=model.Dag}
//	@Router		/workflow/workflow/{id}/dag [get]
func workflowDagDetail(ctx *fiber.Ctx) error {
	id := route.GetIntParam(ctx, "id")

	item, err := store.Database.GetWorkflow(id)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	if len(item.Dag) == 0 {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrNotFound, nil))
	}
	return ctx.JSON(protocol.NewSuccessResponse(item.Dag[0]))
}

// workflow dag update
//
//	@Summary	workflow dag update
//	@Tags		workflow
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int			true	"Workflow ID"
//	@Param		trigger	body		model.Dag	true	"workflow dag data"
//	@Success	200		{object}	protocol.Response
//	@Router		/workflow/workflow/{id}/dag [put]
func workflowDagUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	id := route.GetIntParam(ctx, "id")

	item := new(model.Dag)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	item.UID = uid.String()
	item.Topic = topic
	item.ID = id
	err = store.Database.UpdateDag(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
