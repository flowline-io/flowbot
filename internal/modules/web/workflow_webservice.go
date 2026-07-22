package web

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
	pkgworkflow "github.com/flowline-io/flowbot/pkg/workflow"
)

var workflowWebserviceRules = []webservice.Rule{
	webservice.Get("/workflows", workflowListPage, route.WithNotAuth()),
	webservice.Get("/workflows/list", workflowListTable, route.WithNotAuth()),
	webservice.Put("/workflows/:name/enabled", setWorkflowEnabled, route.WithNotAuth()),
	webservice.Put("/workflows/:name/triggers/:id/enabled", setWorkflowTriggerEnabled, route.WithNotAuth()),
	webservice.Get("/workflows/:name", workflowDetailPage, route.WithNotAuth()),
	webservice.Get("/workflows/:name/runs", workflowRunsPage, route.WithNotAuth()),
	webservice.Get("/workflows/:name/runs/list", workflowRunsTable, route.WithNotAuth()),
	webservice.Get("/workflows/:name/runs/:runID/steps", workflowRunSteps, route.WithNotAuth()),
	webservice.Post("/workflows/:name/run", workflowRunNow, route.WithNotAuth()),
}

func getWorkflowStore() *store.WorkflowStore {
	if store.Database == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewWorkflowStore(client)
}

func getWorkflowRunStore() *store.WorkflowRunStore {
	if store.Database == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewWorkflowRunStore(client)
}

func getWorkflowService() *pkgworkflow.Service {
	if svc := pkgworkflow.ActiveService(); svc != nil {
		return svc
	}
	ws := getWorkflowStore()
	rs := getWorkflowRunStore()
	if ws == nil {
		return nil
	}
	// Local fallback for tests / when server lifecycle has not wired ActiveService.
	return pkgworkflow.NewService(ws, rs, nil, nil)
}

// workflowNameParam returns the decoded :name path parameter for workflow routes.
func workflowNameParam(c fiber.Ctx) (string, error) {
	name, err := decodePathParam(c.Params("name"))
	if err != nil || name == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "invalid workflow name")
	}
	return name, nil
}

func workflowListPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	entries, err := loadWorkflowListEntries(c.Context())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list workflows: %v", err)
	}
	c.Type("html")
	return pages.WorkflowListPage(entries).Render(c.Context(), c.Response().BodyWriter())
}

func workflowListTable(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	entries, err := loadWorkflowListEntries(c.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return renderError(c, "Failed to load workflows")
	}
	c.Type("html")
	return partials.WorkflowListTable(entries).Render(c.Context(), c.Response().BodyWriter())
}

func loadWorkflowListEntries(ctx context.Context) ([]partials.WorkflowListEntry, error) {
	s := getWorkflowStore()
	if s == nil {
		return nil, fmt.Errorf("workflow store not available")
	}
	defs, err := s.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	triggers, err := s.ListTriggers(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if def != nil {
			names = append(names, def.Name)
		}
	}
	lastRuns, err := s.LatestRunStartedAtByNames(ctx, names)
	if err != nil {
		return nil, err
	}
	entries := partials.BuildWorkflowListEntries(defs, triggers, lastRuns)
	since := time.Now().Add(-7 * 24 * time.Hour)
	stats, err := s.RunLatencyStatsByNames(ctx, names, since)
	if err != nil {
		return nil, err
	}
	return partials.AttachWorkflowRunLatencyStats(entries, stats), nil
}

func workflowDetailPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	s := getWorkflowStore()
	if s == nil {
		return types.Errorf(types.ErrInternal, "workflow store not available")
	}
	dto, err := s.GetDefinitionByName(c.Context(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(http.StatusNotFound).SendString("workflow not found")
		}
		return types.Errorf(types.ErrInternal, "get workflow: %v", err)
	}
	meta, err := pkgworkflow.MetadataFromRows(pkgworkflow.WorkflowRows{
		Workflow: dto.Workflow,
		Tasks:    dto.Tasks,
		Triggers: dto.Triggers,
	})
	if err != nil {
		return types.Errorf(types.ErrInternal, "workflow metadata: %v", err)
	}
	yamlBytes, err := pkgworkflow.ExportYAML(meta)
	if err != nil {
		return types.Errorf(types.ErrInternal, "export workflow yaml: %v", err)
	}
	runs, err := s.ListRunsByName(c.Context(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list workflow runs: %v", err)
	}
	if len(runs) > 10 {
		runs = runs[:10]
	}
	c.Type("html")
	return pages.WorkflowDetailPage(meta, dto.Triggers, runs, string(yamlBytes)).Render(c.Context(), c.Response().BodyWriter())
}

func setWorkflowEnabled(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	s := getWorkflowStore()
	if s == nil {
		return types.Errorf(types.ErrInternal, "workflow store not available")
	}
	if _, err := s.SetEnabled(c.Context(), name, body.Enabled); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Workflow not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "set workflow enabled: %v", err)
	}
	if svc := getWorkflowService(); svc != nil {
		if reloadErr := svc.ReloadTriggers(c.Context()); reloadErr != nil {
			flog.Warn("workflow: reload triggers after enabled toggle: %v", reloadErr)
		}
	}
	entries, err := loadWorkflowListEntries(c.Context())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list workflows: %v", err)
	}
	c.Type("html")
	return partials.WorkflowListTable(entries).Render(c.Context(), c.Response().BodyWriter())
}

func setWorkflowTriggerEnabled(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	triggerID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || triggerID <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "invalid trigger id")
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	s := getWorkflowStore()
	if s == nil {
		return types.Errorf(types.ErrInternal, "workflow store not available")
	}
	if _, err := s.SetTriggerEnabled(c.Context(), name, triggerID, body.Enabled); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Workflow trigger not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "set workflow trigger enabled: %v", err)
	}
	if svc := getWorkflowService(); svc != nil {
		if reloadErr := svc.ReloadTriggers(c.Context()); reloadErr != nil {
			flog.Warn("workflow: reload triggers after trigger enabled toggle: %v", reloadErr)
		}
	}
	dto, err := s.GetDefinitionByName(c.Context(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get workflow: %v", err)
	}
	c.Type("html")
	return partials.WorkflowTriggersTable(name, dto.Triggers).Render(c.Context(), c.Response().BodyWriter())
}

func workflowRunsPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	s := getWorkflowStore()
	if s == nil {
		return types.Errorf(types.ErrInternal, "workflow store not available")
	}
	if _, err := s.GetDefinitionByName(c.Context(), name); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(http.StatusNotFound).SendString("workflow not found")
		}
		return types.Errorf(types.ErrInternal, "get workflow: %v", err)
	}
	runs, err := s.ListRunsByName(c.Context(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list workflow runs: %v", err)
	}
	c.Type("html")
	return pages.WorkflowRunsPage(name, runs).Render(c.Context(), c.Response().BodyWriter())
}

func workflowRunsTable(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	s := getWorkflowStore()
	if s == nil {
		c.Status(http.StatusInternalServerError)
		return renderError(c, "Workflow store not available")
	}
	runs, err := s.ListRunsByName(c.Context(), name)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return renderError(c, "Failed to load workflow runs")
	}
	c.Type("html")
	return partials.WorkflowRunsTable(name, runs).Render(c.Context(), c.Response().BodyWriter())
}

func workflowRunSteps(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	runID, err := strconv.ParseInt(c.Params("runID"), 10, 64)
	if err != nil || runID <= 0 {
		return c.Status(http.StatusBadRequest).SendString("invalid run ID")
	}
	rs := getWorkflowRunStore()
	if rs == nil {
		return types.Errorf(types.ErrInternal, "workflow run store not available")
	}
	run, err := rs.GetRun(c.Context(), runID)
	if err != nil {
		if gen.IsNotFound(err) {
			return c.Status(http.StatusNotFound).SendString("run not found")
		}
		return types.Errorf(types.ErrInternal, "get workflow run: %v", err)
	}
	if run == nil || run.WorkflowName != name {
		return c.Status(http.StatusNotFound).SendString("run not found")
	}
	steps, err := rs.GetStepRunsByRunID(c.Context(), runID)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get workflow step runs: %v", err)
	}
	c.Type("html")
	return partials.WorkflowStepRunsDetail(steps).Render(c.Context(), c.Response().BodyWriter())
}

func workflowRunNow(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name, err := workflowNameParam(c)
	if err != nil {
		return err
	}
	s := getWorkflowStore()
	svc := getWorkflowService()
	if s == nil || svc == nil {
		c.Status(http.StatusServiceUnavailable)
		return renderFormError(c, "#form-error", "Workflow run service is not available")
	}
	meta, err := s.GetMetadata(c.Context(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return renderFormError(c, "#form-error", "Workflow not found")
		}
		return types.Errorf(types.ErrInternal, "get workflow metadata: %v", err)
	}
	input, err := parseWorkflowRunInputs(c, meta.Inputs)
	if err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderFormError(c, "#form-error", err.Error())
	}
	input = pkgworkflow.ApplyInputDefaults(meta.Inputs, input)
	runID, err := svc.StartRunAsync(c.Context(), name, "manual", input)
	if err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			c.Status(fiber.StatusUnprocessableEntity)
			msg := err.Error()
			if cause := errors.Unwrap(err); cause != nil {
				msg = cause.Error()
			}
			return renderFormError(c, "#form-error", msg)
		}
		if errors.Is(err, types.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return renderFormError(c, "#form-error", "Workflow not found")
		}
		if errors.Is(err, types.ErrUnavailable) {
			c.Status(http.StatusServiceUnavailable)
			return renderFormError(c, "#form-error", "Workflow run service is not available")
		}
		flog.Error(fmt.Errorf("workflowRunNow: %w", err))
		c.Status(http.StatusInternalServerError)
		return renderFormError(c, "#form-error", "Failed to start workflow run")
	}
	setShowToast(c, "success", fmt.Sprintf("Workflow run #%d started", runID))
	c.Response().Header.Set("HX-Redirect", partials.WorkflowWebPath(name)+"/runs")
	return c.SendStatus(http.StatusOK)
}

// parseWorkflowRunInputs reads run inputs from a JSON body or form fields matching declared inputs.
func parseWorkflowRunInputs(c fiber.Ctx, declared []types.WorkflowInputDef) (types.KV, error) {
	ct := string(c.Request().Header.ContentType())
	if strings.Contains(ct, "application/json") {
		return parseWorkflowRunJSONBody(c.Body(), declared)
	}
	return parseWorkflowRunForm(c, declared)
}

func parseWorkflowRunJSONBody(body []byte, declared []types.WorkflowInputDef) (types.KV, error) {
	if len(body) == 0 {
		return types.KV{}, nil
	}
	var raw map[string]any
	if err := sonic.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}
	if nested, ok := raw["inputs"].(map[string]any); ok {
		raw = nested
	}
	out := types.KV{}
	if len(declared) == 0 {
		maps.Copy(out, raw)
		return out, nil
	}
	for _, def := range declared {
		v, ok := raw[def.Name]
		if !ok || v == nil {
			continue
		}
		coerced, err := coerceWorkflowInputValue(def, v)
		if err != nil {
			return nil, err
		}
		out[def.Name] = coerced
	}
	return out, nil
}

func parseWorkflowRunForm(c fiber.Ctx, declared []types.WorkflowInputDef) (types.KV, error) {
	out := types.KV{}
	for _, def := range declared {
		raw := strings.TrimSpace(c.FormValue(def.Name))
		if raw == "" {
			continue
		}
		coerced, err := coerceWorkflowInputString(def, raw)
		if err != nil {
			return nil, err
		}
		out[def.Name] = coerced
	}
	return out, nil
}

func coerceWorkflowInputValue(def types.WorkflowInputDef, v any) (any, error) {
	switch def.Type {
	case types.WorkflowInputTypeString:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("input %q must be a string", def.Name)
		}
		return s, nil
	case types.WorkflowInputTypeNumber:
		switch n := v.(type) {
		case float64, float32, int, int64, int32:
			return n, nil
		case string:
			return coerceWorkflowInputString(def, n)
		default:
			return nil, fmt.Errorf("input %q must be a number", def.Name)
		}
	case types.WorkflowInputTypeBoolean:
		b, ok := v.(bool)
		if !ok {
			if s, ok := v.(string); ok {
				return coerceWorkflowInputString(def, s)
			}
			return nil, fmt.Errorf("input %q must be a boolean", def.Name)
		}
		return b, nil
	case types.WorkflowInputTypeJSON:
		switch v.(type) {
		case map[string]any, []any, types.KV:
			return v, nil
		case string:
			return coerceWorkflowInputString(def, v.(string))
		default:
			return nil, fmt.Errorf("input %q must be a json object or array", def.Name)
		}
	default:
		return v, nil
	}
}

func coerceWorkflowInputString(def types.WorkflowInputDef, raw string) (any, error) {
	switch def.Type {
	case types.WorkflowInputTypeString, "":
		return raw, nil
	case types.WorkflowInputTypeNumber:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("input %q must be a number", def.Name)
		}
		return f, nil
	case types.WorkflowInputTypeBoolean:
		switch strings.ToLower(raw) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return nil, fmt.Errorf("input %q must be a boolean", def.Name)
		}
	case types.WorkflowInputTypeJSON:
		var decoded any
		if err := sonic.Unmarshal([]byte(raw), &decoded); err != nil {
			return nil, fmt.Errorf("input %q must be valid JSON", def.Name)
		}
		switch decoded.(type) {
		case map[string]any, []any:
			return decoded, nil
		default:
			return nil, fmt.Errorf("input %q must be a json object or array", def.Name)
		}
	default:
		return raw, nil
	}
}
