package web

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var pipelineWebserviceRules = []webservice.Rule{
	webservice.Get("/pipelines", pipelineListPage),
	webservice.Get("/pipelines/list", pipelineListTable),
	webservice.Get("/pipelines/capabilities", getCapabilities),
	webservice.Get("/pipelines/:name", pipelineEditorPage),
	webservice.Post("/pipelines", createPipeline),
	webservice.Put("/pipelines/:name", updatePipelineDraft),
	webservice.Put("/pipelines/:name/publish", publishPipeline),
	webservice.Delete("/pipelines/:name", deletePipeline),
	webservice.Get("/pipelines/:name/yaml", getPipelineYaml),
	webservice.Get("/pipelines/:name/mock", getMockPayload),
	webservice.Post("/pipelines/:name/test", testPipelineStep),
	webservice.Get("/pipelines/:name/runs", pipelineRunsPage),
	webservice.Get("/pipelines/:name/runs/list", pipelineRunsTable),
	webservice.Get("/pipelines/:name/runs/:runID/steps", pipelineRunSteps),
}

func getPipelineDefStore() *store.PipelineStore {
	if store.Database == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewPipelineStore(client)
}

func pipelineListPage(c fiber.Ctx) error {
	s := getPipelineDefStore()
	defs, err := s.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return pages.PipelineListPage(defs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineListTable(c fiber.Ctx) error {
	s := getPipelineDefStore()
	defs, err := s.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(defs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineEditorPage(c fiber.Ctx) error {
	name := c.Params("name")
	c.Type("html")
	return pages.PipelineEditorPage(name).Render(context.Background(), c.Response().BodyWriter())
}

func createPipeline(c fiber.Ctx) error {
	name := c.FormValue("name")
	description := c.FormValue("description")
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "name is required")
	}
	s := getPipelineDefStore()
	if err := s.CreateDefinition(context.Background(), name, description); err != nil {
		if errors.Is(err, types.ErrAlreadyExists) {
			c.Response().Header.Set("HX-Retarget", "#create-form")
			c.Response().Header.Set("HX-Reswap", "beforebegin")
			c.Type("html")
			return c.SendString(fmt.Sprintf(`<div class="bg-red-50 border border-red-200 rounded px-4 py-2 mb-4 text-red-700 text-sm">Pipeline "%s" already exists.</div>`, name))
		}
		return types.Errorf(types.ErrInternal, "create pipeline: %v", err)
	}
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+name)
	return c.SendStatus(200)
}

func updatePipelineDraft(c fiber.Ctx) error {
	name := c.Params("name")
	var body struct {
		Yaml    string `json:"yaml"`
		Version int    `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	s := getPipelineDefStore()
	def, err := s.UpdateDefinitionDraft(context.Background(), name, body.Yaml, body.Version)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return c.Status(409).JSON(fiber.Map{
				"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere. Please refresh the page."},
			})
		}
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "update draft: %v", err)
	}
	return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
}

func publishPipeline(c fiber.Ctx) error {
	name := c.Params("name")
	var body struct {
		Version int `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	s := getPipelineDefStore()

	// Validate YAML structure before publishing
	def, err := s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "publish: get pipeline: %v", err)
	}
	if _, err := pipeline.ParseEditorYAML(def.YamlDraft); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{"code": "VALIDATION_ERROR", "message": "YAML validation failed: " + err.Error()},
		})
	}

	def, err = s.PublishDefinition(context.Background(), name, body.Version)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return c.Status(409).JSON(fiber.Map{
				"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere. Please refresh the page."},
			})
		}
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "publish: %v", err)
	}
	return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
}

func deletePipeline(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	_, err := s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"}})
		}
		return types.Errorf(types.ErrInternal, "delete pipeline: %v", err)
	}
	_, err = s.DeleteDefinitionByName(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "delete pipeline: %v", err)
	}
	// Return refreshed table HTML (HTMX target is #pipeline-list-container)
	defs, err := s.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(defs).Render(context.Background(), c.Response().BodyWriter())
}

func getPipelineYaml(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	def, err := s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get yaml: %v", err)
	}
	return c.JSON(fiber.Map{
		"yaml":    def.YamlDraft,
		"version": def.Version,
		"status":  def.Status,
	})
}

func getMockPayload(c fiber.Ctx) error {
	switch source := c.Query("source"); source {
	case "event":
		return c.JSON(fiber.Map{
			"source": "event",
			"payload": fiber.Map{
				"event_id": "mock-ev-001", "event_type": "item.created",
				"title": "", "entity_id": "", "source": "", "capability": "example", "operation": "create",
			},
			"note": "Generated from event schema. Edit values to match your expected data.",
		})
	case "webhook":
		return c.JSON(fiber.Map{
			"source":  "webhook",
			"payload": fiber.Map{"event_id": "mock-wb-001", "title": "Sample webhook payload", "body": fiber.Map{}},
			"note":    "Edit fields to customize your test data.",
		})
	case "cron":
		return c.JSON(fiber.Map{"source": "cron", "payload": fiber.Map{}, "note": "Cron-triggered pipelines have no event payload."})
	default:
		return types.Errorf(types.ErrInvalidArgument, "missing or invalid source query param")
	}
}

func testPipelineStep(c fiber.Ctx) error {
	var body struct {
		TriggerSource string         `json:"trigger_source"`
		MockPayload   map[string]any `json:"mock_payload"`
		UpToStepIndex int            `json:"up_to_step_index"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	name := c.Params("name")
	s := getPipelineDefStore()
	def, err := s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
	}
	ed, err := pipeline.ParseEditorYAML(def.YamlDraft)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Failed to parse pipeline YAML: " + err.Error()})
	}

	type stepResult struct {
		Name           string         `json:"name"`
		Status         string         `json:"status"`
		DurationMs     int64          `json:"duration_ms,omitempty"`
		Output         map[string]any `json:"output,omitempty"`
		RenderedParams map[string]any `json:"rendered_params,omitempty"`
		Error          string         `json:"error,omitempty"`
	}
	if body.UpToStepIndex < 0 || body.UpToStepIndex >= len(ed.Steps) {
		return c.JSON(fiber.Map{"success": false, "error": "step index out of range"})
	}
	event := types.DataEvent{Data: make(map[string]any)}
	maps.Copy(event.Data, body.MockPayload)
	event.EventID = "mock-test-" + name
	if eid, ok := body.MockPayload["event_id"].(string); ok {
		event.EventID = eid
	}
	if et, ok := body.MockPayload["event_type"].(string); ok {
		event.EventType = et
	}
	rc := pipeline.NewRenderContext(event)
	var results []stepResult
	for i := 0; i <= body.UpToStepIndex; i++ {
		step := ed.Steps[i]
		start := time.Now()
		rendered, rErr := rc.RenderParams(step.Params)
		if rErr != nil {
			results = append(results, stepResult{Name: step.Name, Status: "error", Error: fmt.Sprintf("render params: %v", rErr)})
			return c.JSON(fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results})
		}
		_, iErr := ability.Invoke(context.Background(), step.Capability, step.Operation, rendered)
		duration := time.Since(start).Milliseconds()
		if iErr != nil {
			results = append(results, stepResult{Name: step.Name, Status: "error", Error: fmt.Sprintf("invoke: %v", iErr)})
			return c.JSON(fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results})
		}
		results = append(results, stepResult{
			Name: step.Name, Status: "ok", DurationMs: duration,
			Output: rendered, RenderedParams: rendered,
		})
		rc.RecordStepResult(step.Name, rendered)
	}
	return c.JSON(fiber.Map{"success": true, "steps": results})
}

func pipelineRunsPage(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	runs, err := s.GetRunsByParentName(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return pages.PipelineRunsPage(name, runs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineRunsTable(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	runs, err := s.GetRunsByParentName(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return partials.PipelineRunsTable(name, runs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineRunSteps(c fiber.Ctx) error {
	runID, err := strconv.ParseInt(c.Params("runID"), 10, 64)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid run ID: %v", err)
	}
	s := getPipelineDefStore()
	steps, err := s.GetStepRunsByRunID(context.Background(), runID)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get step runs: %v", err)
	}
	c.Type("html")
	return partials.PipelineStepRunsDetail(steps).Render(context.Background(), c.Response().BodyWriter())
}

// getCapabilities returns all registered capabilities with their operations
// for the pipeline editor capability/operation select dropdowns.
func getCapabilities(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(hub.Default.List()))
}
