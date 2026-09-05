package automate

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var pipelineRules = []webservice.Rule{
	webservice.Post("/apply", applyPipeline),
	webservice.Get("/list", listPipelines),
	webservice.Get("/get/:name", getPipeline),
	webservice.Get("/export/:name", exportPipeline),
	webservice.Delete("/delete/:name", deletePipeline),
	webservice.Post("/run", runPipeline),
	webservice.Get("/runs/:name", listPipelineRuns),
}

func applyPipeline(ctx fiber.Ctx) error {
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
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	result, err := svc.ApplyYAML(ctx.Context(), []byte(yamlText), requestUID(ctx))
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"name":    result.Name,
		"id":      result.ID,
		"enabled": result.Enabled,
		"version": result.Version,
	}))
}

func listPipelines(ctx fiber.Ctx) error {
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	items, err := svc.List(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"pipelines": items}))
}

func getPipeline(ctx fiber.Ctx) error {
	name := pipelineNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	meta, err := svc.Get(ctx.Context(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(meta))
}

func exportPipeline(ctx fiber.Ctx) error {
	name := pipelineNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	yamlText, err := svc.Export(ctx.Context(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"yaml": yamlText}))
}

func deletePipeline(ctx fiber.Ctx) error {
	name := pipelineNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	if err := svc.Delete(ctx.Context(), name); err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"deleted": name}))
}

func runPipeline(ctx fiber.Ctx) error {
	var body struct {
		Name  string         `json:"name"`
		Event map[string]any `json:"event"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	if strings.TrimSpace(body.Name) == "" {
		return types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	runID, err := svc.StartRunAsync(ctx.Context(), body.Name, body.Event, requestUID(ctx))
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusAccepted).JSON(protocol.NewSuccessResponse(types.KV{
		"run_id": runID,
	}))
}

func listPipelineRuns(ctx fiber.Ctx) error {
	name := pipelineNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	svc, err := activePipelineService()
	if err != nil {
		return err
	}
	runs, err := svc.ListRuns(ctx.Context(), name)
	if err != nil {
		return err
	}
	items := make([]types.KV, 0, len(runs))
	for _, r := range runs {
		if r == nil {
			continue
		}
		item := types.KV{
			"id":             r.ID,
			"pipeline_name":  r.PipelineName,
			"status":         r.Status,
			"trigger_source": fmt.Sprint(r.TriggerSource),
			"event_id":       r.EventID,
			"event_type":     r.EventType,
			"created_at":     r.CreatedAt,
			"started_at":     r.StartedAt,
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

func pipelineNameParam(ctx fiber.Ctx) string {
	name := strings.TrimSpace(ctx.Params("name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Query("name"))
	}
	return name
}
