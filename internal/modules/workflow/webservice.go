package workflow

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Post("/apply", applyWorkflow),
	webservice.Get("/list", listWorkflows),
	webservice.Get("/get/:name", getWorkflow),
	webservice.Get("/export/:name", exportWorkflow),
	webservice.Delete("/delete/:name", deleteWorkflow),
	webservice.Post("/run", runWorkflow),
	webservice.Get("/runs/:name", listWorkflowRuns),
}

func applyWorkflow(ctx fiber.Ctx) error {
	var body struct {
		YAML        string `json:"yaml"`
		FileContent string `json:"file_content"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	yamlText := body.YAML
	if yamlText == "" {
		yamlText = body.FileContent
	}
	if strings.TrimSpace(yamlText) == "" {
		return types.Errorf(types.ErrInvalidArgument, "yaml is required")
	}
	svc, err := activeService()
	if err != nil {
		return err
	}
	row, err := svc.ApplyYAML(context.Background(), []byte(yamlText))
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"name":    row.Name,
		"id":      row.ID,
		"enabled": row.Enabled,
	}))
}

func listWorkflows(ctx fiber.Ctx) error {
	svc, err := activeService()
	if err != nil {
		return err
	}
	defs, err := svc.List(context.Background())
	if err != nil {
		return err
	}
	items := make([]types.KV, 0, len(defs))
	for _, d := range defs {
		if d == nil {
			continue
		}
		items = append(items, types.KV{
			"id":              d.ID,
			"name":            d.Name,
			"describe":        d.Describe,
			"enabled":         d.Enabled,
			"resumable":       d.Resumable,
			"max_concurrency": d.MaxConcurrency,
		})
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"workflows": items}))
}

func getWorkflow(ctx fiber.Ctx) error {
	name := workflowNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "workflow name is required")
	}
	svc, err := activeService()
	if err != nil {
		return err
	}
	meta, err := svc.Get(context.Background(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(meta))
}

func exportWorkflow(ctx fiber.Ctx) error {
	name := workflowNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "workflow name is required")
	}
	svc, err := activeService()
	if err != nil {
		return err
	}
	data, err := svc.Export(context.Background(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"yaml": string(data)}))
}

func deleteWorkflow(ctx fiber.Ctx) error {
	name := workflowNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "workflow name is required")
	}
	svc, err := activeService()
	if err != nil {
		return err
	}
	if err := svc.Delete(context.Background(), name); err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"deleted": name}))
}

func runWorkflow(ctx fiber.Ctx) error {
	var body struct {
		Name  string   `json:"name"`
		Input types.KV `json:"input"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	if strings.TrimSpace(body.Name) == "" {
		return types.Errorf(types.ErrInvalidArgument, "workflow name is required")
	}
	svc, err := activeService()
	if err != nil {
		return err
	}
	runID, err := svc.StartRunAsync(context.Background(), body.Name, "manual", body.Input)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusAccepted).JSON(protocol.NewSuccessResponse(types.KV{
		"run_id": runID,
	}))
}

func listWorkflowRuns(ctx fiber.Ctx) error {
	name := workflowNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "workflow name is required")
	}
	svc, err := activeService()
	if err != nil {
		return err
	}
	runs, err := svc.ListRuns(context.Background(), name)
	if err != nil {
		return err
	}
	items := make([]types.KV, 0, len(runs))
	for _, r := range runs {
		if r == nil {
			continue
		}
		item := types.KV{
			"id":            r.ID,
			"workflow_name": r.WorkflowName,
			"status":        r.Status,
			"trigger_type":  r.TriggerType,
			"created_at":    r.CreatedAt,
			"started_at":    r.StartedAt,
		}
		if r.CompletedAt != nil {
			item["completed_at"] = *r.CompletedAt
		}
		if r.Error != "" {
			item["error"] = r.Error
		}
		items = append(items, item)
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"runs": items}))
}

func workflowNameParam(ctx fiber.Ctx) string {
	name := strings.TrimSpace(ctx.Params("name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Query("name"))
	}
	return name
}
