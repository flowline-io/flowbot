package web

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
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
	webservice.Get("/pipelines/stats", pipelineStats),
	webservice.Get("/pipelines/:name", pipelineEditorPage),
	webservice.Post("/pipelines", createPipeline),
	webservice.Put("/pipelines/:name", updatePipelineDraft),
	webservice.Put("/pipelines/:name/publish", publishPipeline),
	webservice.Put("/pipelines/:name/enabled", setPipelineEnabled),
	webservice.Delete("/pipelines/:name", deletePipeline),
	webservice.Get("/pipelines/:name/yaml", getPipelineYaml),
	webservice.Get("/pipelines/:name/mock", getMockPayload),
	webservice.Post("/pipelines/:name/test", testPipelineStep),
	webservice.Get("/pipelines/:name/runs", pipelineRunsPage),
	webservice.Get("/pipelines/:name/runs/list", pipelineRunsTable),
	webservice.Get("/pipelines/:name/runs/:runID/steps", pipelineRunSteps),
	webservice.Get("/pipelines/:name/runs/:runID/live", pipelineRunLivePage),
	webservice.Get("/pipelines/:name/runs/:runID/live/watch", watchPipelineRunLive),
	webservice.Get("/pipelines/:name/stats", pipelineStats),
	webservice.Get("/pipelines/:name/versions", listPipelineVersions),
	webservice.Get("/pipelines/:name/versions/:version", getPipelineVersion),
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
	entries, err := buildPipelineListEntries(context.Background(), s, defs)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipeline last runs: %v", err)
	}
	c.Type("html")
	return pages.PipelineListPage(entries).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineListTable(c fiber.Ctx) error {
	s := getPipelineDefStore()
	defs, err := s.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	entries, err := buildPipelineListEntries(context.Background(), s, defs)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipeline last runs: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(entries).Render(context.Background(), c.Response().BodyWriter())
}

// buildPipelineListEntries loads last-run timestamps and latency stats and builds pipeline list rows.
func buildPipelineListEntries(ctx context.Context, s *store.PipelineStore, defs []*gen.PipelineDefinition) ([]partials.PipelineListEntry, error) {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if def != nil {
			names = append(names, def.Name)
		}
	}
	lastRuns, err := s.LatestRunStartedAtByParentNames(ctx, names)
	if err != nil {
		return nil, err
	}
	entries := partials.BuildPipelineListEntries(defs, lastRuns)
	since := time.Now().Add(-7 * 24 * time.Hour)
	stats, err := s.RunLatencyStatsByParentNames(ctx, names, since)
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
	c.Type("html")
	return pages.PipelineEditorPage(name).Render(context.Background(), c.Response().BodyWriter())
}

func createPipeline(c fiber.Ctx) error {
	name := c.FormValue("name")
	description := c.FormValue("description")
	if err := pipeline.ValidateName(name); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderFormError(c, "#form-error", err.Error())
	}
	s := getPipelineDefStore()
	if err := s.CreateDefinition(context.Background(), name, description, getUID(c)); err != nil {
		if errors.Is(err, types.ErrAlreadyExists) {
			c.Status(fiber.StatusUnprocessableEntity)
			return renderFormError(c, "#form-error", fmt.Sprintf("Pipeline %q already exists.", name))
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
	// Backfill owner for pipelines created before created_by existed.
	if err := s.EnsureDefinitionCreatedBy(context.Background(), name, getUID(c)); err != nil {
		flog.Error(fmt.Errorf("ensure pipeline created_by before publish: %w", err))
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
	if reloadErr := pipeline.ReloadDefinitions(context.Background()); reloadErr != nil {
		flog.Error(fmt.Errorf("reload pipeline engine after publish: %w", reloadErr))
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
	s := getPipelineDefStore()
	_, err = s.SetDefinitionEnabled(context.Background(), name, body.Enabled)
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
		return types.Errorf(types.ErrInternal, "set pipeline enabled: %v", err)
	}
	if reloadErr := pipeline.ReloadDefinitions(context.Background()); reloadErr != nil {
		return types.Errorf(types.ErrInternal, "reload pipeline engine after enabled toggle: %v", reloadErr)
	}
	defs, err := s.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	entries, err := buildPipelineListEntries(context.Background(), s, defs)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipeline last runs: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(entries).Render(context.Background(), c.Response().BodyWriter())
}

func deletePipeline(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	s := getPipelineDefStore()
	_, err = s.GetDefinitionByName(context.Background(), name)
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
	entries, err := buildPipelineListEntries(context.Background(), s, defs)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipeline last runs: %v", err)
	}
	c.Type("html")
	return partials.PipelineListTable(entries).Render(context.Background(), c.Response().BodyWriter())
}

func getPipelineYaml(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
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

func listPipelineVersions(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	s := getPipelineDefStore()
	// Verify pipeline exists first, since ListDefinitionVersions does not
	// return ErrNotFound for an unknown pipeline name.
	_, err = s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "list versions: %v", err)
	}
	vers, err := s.ListDefinitionVersions(context.Background(), name)
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
	s := getPipelineDefStore()
	ver, err := s.GetDefinitionVersion(context.Background(), name, version)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get version: %v", err)
	}
	return c.JSON(fiber.Map{
		"yaml":       ver.Yaml,
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
	rc := pipeline.NewRenderContext(buildPipelineTestEvent(c, name, body.MockPayload))
	var results []stepResult
	for i := 0; i <= body.UpToStepIndex; i++ {
		step := ed.Steps[i]
		start := time.Now()
		rendered, rErr := rc.RenderParams(step.Params)
		if rErr != nil {
			results = append(results, stepResult{Name: step.Name, Status: "error", Error: fmt.Sprintf("render params: %v", rErr)})
			return c.JSON(fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results})
		}
		pipeline.InjectAgentRunDefaults(step, rendered, rc, name)
		res, iErr := capability.Invoke(context.Background(), step.Capability, step.Operation, rendered)
		duration := time.Since(start).Milliseconds()
		if iErr != nil {
			results = append(results, stepResult{Name: step.Name, Status: "error", Error: fmt.Sprintf("invoke: %v", iErr)})
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
	s := getPipelineDefStore()
	runs, err := s.GetRunsByParentName(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return pages.PipelineRunsPage(name, runs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineRunsTable(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
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

// watchPipelineRunLive opens an SSE stream for a running pipeline.
func watchPipelineRunLive(c fiber.Ctx) error {
	runIDParam := c.Params("runID")
	runID, err := strconv.ParseInt(runIDParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid runID")
	}
	stream := pipeline.StreamName(runID)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	ctx := c.Context()
	redisClient := rdb.Client
	if redisClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("redis not available")
	}

	return c.SendStreamWriter(func(w *bufio.Writer) {
		lastID := "0"
		for {
			select {
			case <-ctx.Done():
				return
			default:
				result, err := redisClient.XRead(ctx, broadcastStreamReadArgs(stream, lastID)).Result()
				if done := handleStreamRead(w, result, err, &lastID); done {
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
		if done := writeSSEEvent(w, data); done {
			return true
		}
	}
	return false
}

func writeHeartbeat(w *bufio.Writer) bool {
	if _, fErr := fmt.Fprintf(w, ": heartbeat\n\n"); fErr != nil {
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
	var evt pipeline.StepProgressEvent
	if err := sonic.UnmarshalString(data, &evt); err != nil {
		return false
	}
	return evt.StepIndex == -1 &&
		(evt.Status == "complete" || evt.Status == "failed")
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
		return c.Status(fiber.StatusBadRequest).SendString("invalid runID")
	}

	s := getPipelineDefStore()
	if s == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("store not available")
	}

	run, err := s.GetRunByID(context.Background(), runID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("run not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load run")
	}

	steps, err := s.ListStepRunsByRunID(context.Background(), runID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to load steps")
	}

	initSteps := make([]pages.StepState, len(steps))
	for i, s := range steps {
		ss := pages.StepState{
			Name:   s.StepName,
			Status: stepRunStatusLabel(s.Status),
			Output: s.Result,
			Error:  s.Error,
			Input:  s.Params,
		}
		if s.CompletedAt != nil && !s.StartedAt.IsZero() {
			ss.ElapsedMs = s.CompletedAt.Sub(s.StartedAt).Milliseconds()
		}
		initSteps[i] = ss
	}

	c.Type("html")
	return pages.PipelineRunLivePage(pages.PipelineRunLiveParams{
		RunID:        runID,
		PipelineName: pipelineName,
		Trigger:      run.EventType,
		TotalSteps:   len(steps),
		RunStatus:    pipelineRunStatusLabel(run.Status),
		Steps:        initSteps,
	}).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineStats(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	sinceStr := c.Query("since", "")
	since := time.Time{}
	if sinceStr != "" {
		parsed, err := time.Parse("2006-01-02", sinceStr)
		if err != nil {
			return types.Errorf(types.ErrInvalidArgument, "invalid since date: %v", err)
		}
		since = parsed
	}
	groupBy := c.Query("groupBy", "day")
	if groupBy != "day" && groupBy != "week" && groupBy != "month" {
		return types.Errorf(types.ErrInvalidArgument, "groupBy must be day, week, or month")
	}

	s := getPipelineDefStore()
	if s == nil {
		return types.Errorf(types.ErrInternal, "store not available")
	}
	if name != "" {
		_, err = s.GetDefinitionByName(context.Background(), name)
		if err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return types.Errorf(types.ErrNotFound, "pipeline %s not found", name)
			}
			return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
		}
	}

	stats, err := s.PipelineStats(context.Background(), name, since, groupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "pipeline stats: %v", err)
	}

	accept := c.Get("Accept", "")
	if accept == "application/json" {
		return c.JSON(stats)
	}
	c.Type("html")
	return partials.PipelineStats(name, stats).Render(context.Background(), c.Response().BodyWriter())
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
