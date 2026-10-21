package web

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var pipelineWebserviceRules = []webservice.Rule{
	webservice.Get("/pipelines", pipelineListPage),
	webservice.Get("/pipelines/list", pipelineListTable),
	webservice.Get("/pipelines/capabilities", getCapabilities),
	webservice.Get("/pipelines/agent-run-options", getAgentRunOptions),
	webservice.Get("/pipelines/function-invoke-options", getFunctionInvokeOptions),
	webservice.Get("/pipelines/stats", pipelineStats),
	webservice.Get("/pipelines/:name", pipelineEditorPage),
	webservice.Post("/pipelines", createPipeline),
	webservice.Put("/pipelines/:name", updatePipelineDraft),
	webservice.Put("/pipelines/:name/rename", renamePipeline),
	webservice.Put("/pipelines/:name/publish", publishPipeline),
	webservice.Put("/pipelines/:name/enabled", setPipelineEnabled),
	webservice.Delete("/pipelines/:name", deletePipeline),
	webservice.Get("/pipelines/:name/yaml", getPipelineYaml),
	webservice.Get("/pipelines/:name/mock", getMockPayload),
	webservice.Post("/pipelines/:name/test", testPipelineStep),
	webservice.Get("/pipelines/:name/runs", pipelineRunsPage),
	webservice.Get("/pipelines/:name/runs/list", pipelineRunsTable),
	webservice.Get("/pipelines/:name/runs/:runID/steps", pipelineRunSteps),
	webservice.Post("/pipelines/:name/runs/:runID/retry", retryPipelineRun),
	webservice.Get("/pipelines/:name/runs/:runID/live", pipelineRunLivePage),
	webservice.Get("/pipelines/:name/runs/:runID/live/watch", watchPipelineRunLive),
	webservice.Get("/pipelines/:name/stats", pipelineStats),
	webservice.Get("/pipelines/:name/versions", listPipelineVersions),
	webservice.Get("/pipelines/:name/versions/:version", getPipelineVersion),
}

func pipelineSvc() (*pipeline.Service, error) {
	svc := pipeline.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	return svc, nil
}

func pipelineListPage(c fiber.Ctx) error {
	entries, err := loadPipelineListEntries(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return pages.PipelineListPage(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func pipelineListTable(c fiber.Ctx) error {
	entries, err := loadPipelineListEntries(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func loadPipelineListEntries(ctx context.Context) ([]partials.PipelineListEntry, error) {
	svc, err := pipelineSvc()
	if err != nil {
		return nil, err
	}
	defs, err := svc.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if def != nil {
			names = append(names, def.Name)
		}
	}
	lastRuns, err := svc.LatestRunStartedAt(ctx, names)
	if err != nil {
		return nil, err
	}
	entries := partials.BuildPipelineListEntries(derefSlice(defs), lastRuns)
	since := time.Now().Add(-7 * 24 * time.Hour)
	stats, err := svc.RunLatencyStats(ctx, names, since)
	if err != nil {
		return nil, err
	}
	return partials.AttachRunLatencyStats(entries, stats), nil
}

func pipelineEditorPage(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	row, err := svc.GetDefinition(context.Background(), name)
	if err != nil {
		return err
	}
	def := model.PipelineDefinition{}
	if row != nil {
		def = *row
	}
	if def.LinkedBlueprint() {
		return renderLinkedPipelinePage(c, def)
	}
	c.Type("html")
	return pages.PipelineEditorPage(c.Context(), name).Render(c.Context(), c.Response().BodyWriter())
}

func createPipeline(c fiber.Ctx) error {
	name := c.FormValue("name")
	description := c.FormValue("description")
	if err := pipeline.ValidateName(name); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderFormErrorKey(c, "#form-error", "error.validation")
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	if err := svc.Create(context.Background(), name, description, getUID(c)); err != nil {
		if errors.Is(err, types.ErrAlreadyExists) {
			c.Status(fiber.StatusUnprocessableEntity)
			return renderFormError(c, "#form-error", webMsgData(c, "error.pipeline.already_exists", map[string]any{"Name": name}))
		}
		return types.Errorf(types.ErrInternal, "create pipeline: %v", err)
	}
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+url.PathEscape(name))
	return c.SendStatus(200)
}

func updatePipelineDraft(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	var body struct {
		Yaml    string `json:"yaml"`
		Version int    `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	existing, err := svc.GetDefinition(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
	}
	if existing != nil && existing.BlueprintID != "" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fiber.Map{"code": "LINKED_BLUEPRINT", "message": webMsg(c, "error.blueprint.linked_readonly")},
		})
	}
	def, err := svc.SaveDraft(context.Background(), name, body.Yaml, body.Version)
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

// renamePipeline renames a pipeline definition and redirects clients to the new URL.
func renamePipeline(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	newName := strings.TrimSpace(body.Name)
	if err := pipeline.ValidateName(newName); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": fiber.Map{"code": "VALIDATION_ERROR", "message": webMsg(c, "error.validation")},
		})
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	def, err := svc.Rename(context.Background(), name, newName)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		if errors.Is(err, types.ErrAlreadyExists) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "ALREADY_EXISTS", "message": fmt.Sprintf("Pipeline %q already exists.", newName)},
			})
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "VALIDATION_ERROR", "message": webMsg(c, "error.validation")},
			})
		}
		return types.Errorf(types.ErrInternal, "rename pipeline: %v", err)
	}
	return c.JSON(fiber.Map{
		"name":    def.Name,
		"version": def.Version,
		"status":  def.Status,
	})
}

func publishPipeline(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	var body struct {
		Version int `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	def, err := svc.Publish(context.Background(), name, body.Version, getUID(c))
	if err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{"code": "VALIDATION_ERROR", "message": webMsg(c, "error.pipeline.yaml_validation_failed")},
			})
		}
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

func setPipelineEnabled(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	err = svc.SetEnabled(context.Background(), name, body.Enabled)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{"code": "INVALID_ARGUMENT", "message": "Only published pipelines can be paused or resumed"},
			})
		}
		return err
	}
	entries, err := loadPipelineListEntries(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func deletePipeline(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	if err = svc.Delete(context.Background(), name); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"}})
		}
		return types.Errorf(types.ErrInternal, "delete pipeline: %v", err)
	}
	entries, err := loadPipelineListEntries(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func getPipelineYaml(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	def, err := svc.GetDefinition(context.Background(), name)
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

func listPipelineVersions(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	// Verify pipeline exists first, since listing versions does not
	// return ErrNotFound for an unknown pipeline name.
	_, err = svc.GetDefinition(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "list versions: %v", err)
	}
	vers, err := svc.ListVersions(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list versions: %v", err)
	}
	items := make([]fiber.Map, 0, len(vers))
	for _, v := range vers {
		items = append(items, fiber.Map{
			"version":    v.Version,
			"created_at": v.CreatedAt,
		})
	}
	return c.JSON(items)
}

func getPipelineVersion(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	version, err := strconv.Atoi(c.Params("version"))
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid version: %v", err)
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	ver, err := svc.GetVersion(context.Background(), name, version)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get version: %v", err)
	}
	return c.JSON(fiber.Map{
		"yaml":       ver.YAML,
		"version":    ver.Version,
		"created_at": ver.CreatedAt,
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

// buildPipelineTestEvent constructs a DataEvent for pipeline step testing.
func buildPipelineTestEvent(c fiber.Ctx, name string, payload map[string]any) types.DataEvent {
	event := types.DataEvent{Data: make(map[string]any)}
	maps.Copy(event.Data, payload)
	event.EventID = "mock-test-" + name
	if eid, ok := payload["event_id"].(string); ok {
		event.EventID = eid
	}
	if et, ok := payload["event_type"].(string); ok {
		event.EventType = et
	}
	if uid, ok := payload["uid"].(string); ok && strings.TrimSpace(uid) != "" {
		event.UID = strings.TrimSpace(uid)
	} else if uid, err := webUID(c); err == nil {
		event.UID = uid.String()
	}
	return event
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
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	def, err := svc.GetDefinition(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
	}
	ed, err := pipeline.ParseEditorYAML(def.YamlDraft)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": webMsg(c, "error.pipeline.parse_yaml_failed")})
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
	rc := pipeline.NewRenderContext(buildPipelineTestEvent(c, name, body.MockPayload))
	var results []stepResult
	for i := 0; i <= body.UpToStepIndex; i++ {
		step := ed.Steps[i]
		start := time.Now()
		rendered, rErr := rc.RenderParams(step.Params)
		if rErr != nil {
			results = append(results, stepResult{Name: step.Name, Status: "error", DurationMs: time.Since(start).Milliseconds(), Error: fmt.Sprintf("render params: %v", rErr)})
			return c.JSON(fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results})
		}
		pipeline.InjectAgentRunDefaults(step, rendered, rc, name)
		res, iErr := capability.Invoke(context.Background(), step.Capability, step.Operation, rendered)
		duration := time.Since(start).Milliseconds()
		if iErr != nil {
			results = append(results, stepResult{Name: step.Name, Status: "error", DurationMs: duration, Error: fmt.Sprintf("invoke: %v", iErr)})
			return c.JSON(fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results})
		}
		stepOutput := pipeline.StepResultFromInvoke(res)
		results = append(results, stepResult{
			Name: step.Name, Status: "ok", DurationMs: duration,
			Output: stepOutput, RenderedParams: rendered,
		})
		rc.RecordStepResult(step.Name, stepOutput)
	}
	return c.JSON(fiber.Map{"success": true, "steps": results})
}

func pipelineRunsPage(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	runs, err := svc.ListRuns(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return pages.PipelineRunsPage(c.Context(), name, derefSlice(runs)).Render(c.Context(), c.Response().BodyWriter())
}

func pipelineRunsTable(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	runs, err := svc.ListRuns(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return partials.PipelineRunsTable(c.Context(), name, derefSlice(runs)).Render(c.Context(), c.Response().BodyWriter())
}

func pipelineRunSteps(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	runID, err := strconv.ParseInt(c.Params("runID"), 10, 64)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid run ID: %v", err)
	}
	svc, err := pipelineSvc()
	if err != nil {
		return err
	}
	run, err := svc.GetRun(context.Background(), runID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.Errorf(types.ErrNotFound, "pipeline run %d", runID)
		}
		return types.Errorf(types.ErrInternal, "get run: %v", err)
	}
	if run == nil || !pipeline.RunBelongsToParent(run.PipelineName, name) {
		return types.Errorf(types.ErrNotFound, "pipeline run %d", runID)
	}
	steps, err := svc.ListStepRunsByID(context.Background(), runID)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get step runs: %v", err)
	}
	retry := partials.PipelineRunRetry{
		PipelineName: name,
		RunID:        runID,
		Enabled:      run.Status == int(types.PipelineFailed),
	}
	c.Type("html")
	return partials.PipelineStepRunsDetail(c.Context(), derefSlice(steps), retry).Render(c.Context(), c.Response().BodyWriter())
}

func retryPipelineRun(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	runID, err := strconv.ParseInt(c.Params("runID"), 10, 64)
	if err != nil {
		return toastErrorKey(c, "error.pipeline.invalid_run_id")
	}
	svc, err := pipelineSvc()
	if err != nil {
		return toastErrorKey(c, "toast.pipeline.retry_failed")
	}
	if err := svc.Retry(c.Context(), name, runID); err != nil {
		if errors.Is(err, pipeline.ErrRetryEventMissing) {
			return toastErrorKey(c, "toast.pipeline.retry_event_missing")
		}
		if errors.Is(err, pipeline.ErrRetryNotFailed) ||
			errors.Is(err, pipeline.ErrRetryDefMissing) ||
			errors.Is(err, pipeline.ErrRetryStepMissing) ||
			errors.Is(err, types.ErrConflict) {
			return toastRetryPipelineError(c, err)
		}
		if errors.Is(err, types.ErrNotFound) {
			return toastErrorKey(c, "error.pipeline.run_not_found")
		}
		if errors.Is(err, types.ErrUnavailable) {
			flog.Error(fmt.Errorf("retry pipeline run %d: %w", runID, err))
			return toastErrorKey(c, "toast.pipeline.retry_failed")
		}
		return toastRetryPipelineError(c, err)
	}
	c.Response().Header.Set("HX-Redirect", partials.PipelineRunLivePath(name, runID))
	return c.SendStatus(http.StatusOK)
}

func toastRetryPipelineError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, pipeline.ErrRetryNotFailed):
		return toastErrorKey(c, "toast.pipeline.retry_not_failed")
	case errors.Is(err, pipeline.ErrRetryDefMissing):
		return toastErrorKey(c, "toast.pipeline.retry_unpublished")
	case errors.Is(err, pipeline.ErrRetryStepMissing):
		return toastErrorKey(c, "toast.pipeline.retry_step_missing")
	case errors.Is(err, types.ErrConflict):
		return toastErrorKey(c, "toast.pipeline.retry_conflict")
	default:
		flog.Error(fmt.Errorf("retry pipeline run: %w", err))
		return toastErrorKey(c, "toast.pipeline.retry_failed")
	}
}

// getCapabilities returns all registered capabilities with their operations
// for the pipeline editor capability/operation select dropdowns.
func getCapabilities(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(hub.Default.List()))
}

type agentRunOptionsResponse struct {
	Tools  []string                         `json:"tools"`
	Skills []model.AgentSubagentSkillOption `json:"skills"`
}

// getAgentRunOptions returns selectable tools and skills for pipeline agent.run steps.
func getAgentRunOptions(ctx fiber.Ctx) error {
	skills, err := listAgentSubagentSkillOptions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list agent skills: %v", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(agentRunOptionsResponse{
		Tools:  chatagent.SelectableSubagentTools(),
		Skills: skills,
	}))
}

type functionInvokeOption struct {
	Name              string `json:"name"`
	PublishedVersions []int  `json:"published_versions"`
	LatestVersion     int    `json:"latest_version,omitempty"`
}

type functionInvokeOptionsResponse struct {
	Functions []functionInvokeOption `json:"functions"`
}

// getFunctionInvokeOptions returns published functions and versions for pipeline functions.invoke steps.
func getFunctionInvokeOptions(c fiber.Ctx) error {
	svc := getFunctionService()
	if svc == nil {
		return c.JSON(protocol.NewSuccessResponse(functionInvokeOptionsResponse{
			Functions: []functionInvokeOption{},
		}))
	}
	reqCtx := c.Context()
	items, err := svc.ListAll(reqCtx)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list functions: %v", err)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if item.PublishedVersion != nil {
			names = append(names, item.Name)
		}
	}
	versionLists := map[string][]int{}
	if len(names) > 0 {
		listed, listErr := svc.PublishedVersionNumbers(reqCtx, names)
		if listErr != nil {
			flog.Error(fmt.Errorf("list function published versions: %w", listErr))
		} else {
			versionLists = listed
		}
	}
	options := make([]functionInvokeOption, 0, len(names))
	for _, item := range items {
		if item.PublishedVersion == nil {
			continue
		}
		versions := versionLists[item.Name]
		if len(versions) == 0 {
			versions = []int{*item.PublishedVersion}
		}
		options = append(options, functionInvokeOption{
			Name:              item.Name,
			PublishedVersions: versions,
			LatestVersion:     versions[0],
		})
	}
	return c.JSON(protocol.NewSuccessResponse(functionInvokeOptionsResponse{
		Functions: options,
	}))
}

// watchPipelineRunLive opens an SSE stream for a running pipeline.
func watchPipelineRunLive(c fiber.Ctx) error {
	runIDParam := c.Params("runID")
	runID, err := strconv.ParseInt(runIDParam, 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(webMsg(c, "error.pipeline.invalid_run_id"))
	}
	stream := pipeline.StreamName(runID)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	ctx := c.Context()
	redisClient := rdb.Client
	if redisClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString(webMsg(c, "error.pipeline.redis_unavailable"))
	}

	return c.SendStreamWriter(func(w *bufio.Writer) {
		lastID := "0"
		for {
			select {
			case <-ctx.Done():
				return
			default:
				result, err := redisClient.XRead(ctx, broadcastStreamReadArgs(stream, lastID)).Result()
				if handleStreamRead(w, result, err, &lastID) {
					return
				}
			}
		}
	})
}

func broadcastStreamReadArgs(stream, lastID string) *redis.XReadArgs {
	return &redis.XReadArgs{
		Streams: []string{stream, lastID},
		Count:   10,
		Block:   5 * time.Second,
	}
}

func handleStreamRead(w *bufio.Writer, result []redis.XStream, err error, lastID *string) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	if err == redis.Nil || len(result) == 0 {
		return writeHeartbeat(w)
	}
	if err != nil {
		time.Sleep(2 * time.Second)
		return false
	}
	for _, msg := range result[0].Messages {
		*lastID = msg.ID
		data, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}
		if writeSSEEvent(w, data) {
			return true
		}
	}
	return false
}

func writeHeartbeat(w *bufio.Writer) bool {
	if _, fErr := fmt.Fprint(w, ": heartbeat\n\n"); fErr != nil {
		return true
	}
	return w.Flush() != nil
}

func writeSSEEvent(w *bufio.Writer, data string) bool {
	if _, fErr := fmt.Fprintf(w, "data: %s\n\n", data); fErr != nil {
		return true
	}
	if fErr := w.Flush(); fErr != nil {
		return true
	}
	return pipeline.IsTerminalProgressJSON(data)
}

// pipelineRunLivePage renders the live run dashboard page.
func pipelineRunLivePage(c fiber.Ctx) error {
	pipelineName, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	runIDParam := c.Params("runID")
	runID, err := strconv.ParseInt(runIDParam, 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(webMsg(c, "error.pipeline.invalid_run_id"))
	}

	svc, err := pipelineSvc()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(webMsg(c, "empty.store_unavailable"))
	}

	run, err := svc.GetRun(context.Background(), runID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString(webMsg(c, "error.pipeline.run_not_found"))
		}
		return c.Status(fiber.StatusInternalServerError).SendString(webMsg(c, "error.pipeline.load_run_failed"))
	}

	steps, err := svc.ListStepRuns(context.Background(), runID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(webMsg(c, "error.pipeline.load_steps_failed"))
	}

	initSteps := make([]pages.StepState, len(steps))
	for i, step := range steps {
		if step == nil {
			continue
		}
		ss := pages.StepState{
			Name:   step.StepName,
			Status: stepRunStatusLabel(step.Status),
			Output: step.Result,
			Error:  step.Error,
			Input:  step.Params,
		}
		if step.CompletedAt != nil && !step.StartedAt.IsZero() {
			ss.ElapsedMs = step.CompletedAt.Sub(step.StartedAt).Milliseconds()
		}
		initSteps[i] = ss
	}

	c.Type("html")
	return pages.PipelineRunLivePage(c.Context(), pages.PipelineRunLiveParams{
		RunID:        runID,
		PipelineName: pipelineName,
		Trigger:      run.EventType,
		TotalSteps:   len(steps),
		RunStatus:    pipelineRunStatusLabel(run.Status),
		Steps:        initSteps,
	}).Render(c.Context(), c.Response().BodyWriter())
}

func pipelineStats(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	since, tabs, err := parseStatsTabQuery(c)
	if err != nil {
		return err
	}

	svc, err := pipelineSvc()
	if err != nil {
		return types.Errorf(types.ErrInternal, "pipeline service not ready")
	}
	if name != "" {
		_, err = svc.GetDefinition(context.Background(), name)
		if err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return types.Errorf(types.ErrNotFound, "pipeline %s not found", name)
			}
			return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
		}
	}

	stats, err := svc.Stats(context.Background(), name, since, time.Time{}, tabs.GroupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "pipeline stats: %v", err)
	}

	accept := c.Get("Accept", "")
	if accept == "application/json" {
		return c.JSON(stats)
	}
	c.Type("html")
	return partials.PipelineStats(c.Context(), name, stats, tabs).Render(c.Context(), c.Response().BodyWriter())
}

// stepRunStatusLabel converts an ent PipelineStepRun status int to a display string.
func stepRunStatusLabel(status int) string {
	switch status {
	case 1:
		return "running"
	case 2:
		return "done"
	case 4:
		return "error"
	default:
		return "pending"
	}
}

// pipelineRunStatusLabel converts an ent PipelineRun status int to a display string.
func pipelineRunStatusLabel(status int) string {
	switch status {
	case 1:
		return "running"
	case 2:
		return "done"
	case 4:
		return "failed"
	default:
		return "pending"
	}
}
