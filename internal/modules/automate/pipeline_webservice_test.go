package automate

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgpipeline "github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type pipelineHandlerCatalog struct {
	defs map[string]*model.PipelineDefinition
	runs map[string][]*model.PipelineRun
}

func newpipelineHandlerCatalog() *pipelineHandlerCatalog {
	return &pipelineHandlerCatalog{
		defs: map[string]*model.PipelineDefinition{},
		runs: map[string][]*model.PipelineRun{},
	}
}

func (c *pipelineHandlerCatalog) CreateDefinition(_ context.Context, name, description, createdBy string) error {
	if _, ok := c.defs[name]; ok {
		return types.ErrAlreadyExists
	}
	c.defs[name] = &model.PipelineDefinition{
		ID: 1, Name: name, Description: description, Version: 1, Status: "draft", CreatedBy: createdBy,
	}
	return nil
}

func (c *pipelineHandlerCatalog) GetDefinitionByName(_ context.Context, name string) (*model.PipelineDefinition, error) {
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *def
	return &cp, nil
}

func (c *pipelineHandlerCatalog) UpdateDefinitionDraft(_ context.Context, name, yamlDraft string, version int) (*model.PipelineDefinition, error) {
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	if def.Version != version {
		return nil, types.ErrConflict
	}
	def.YamlDraft = yamlDraft
	def.Version++
	cp := *def
	return &cp, nil
}

func (c *pipelineHandlerCatalog) PublishDefinition(_ context.Context, name string, version int) (*model.PipelineDefinition, error) {
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	if def.Version != version {
		return nil, types.ErrConflict
	}
	published := def.YamlDraft
	def.YamlPublished = &published
	def.Status = "published"
	def.Version++
	cp := *def
	return &cp, nil
}

func (c *pipelineHandlerCatalog) EnsureDefinitionCreatedBy(_ context.Context, name, createdBy string) error {
	def, ok := c.defs[name]
	if !ok {
		return types.ErrNotFound
	}
	if def.CreatedBy == "" {
		def.CreatedBy = createdBy
	}
	return nil
}

func (c *pipelineHandlerCatalog) DeleteDefinitionByName(_ context.Context, name string) (int64, error) {
	if _, ok := c.defs[name]; !ok {
		return 0, nil
	}
	n := int64(len(c.runs[name]))
	delete(c.defs, name)
	delete(c.runs, name)
	return n, nil
}

func (c *pipelineHandlerCatalog) ListPublishedDefinitions(context.Context) ([]pkgpipeline.DefinitionRecord, error) {
	out := make([]pkgpipeline.DefinitionRecord, 0, len(c.defs))
	for _, def := range c.defs {
		if def.YamlPublished == nil {
			continue
		}
		out = append(out, pkgpipeline.DefinitionRecord{
			Name: def.Name, Description: def.Description, YAML: *def.YamlPublished,
		})
	}
	return out, nil
}

func (c *pipelineHandlerCatalog) GetRunsByParentName(_ context.Context, parentName string) ([]*model.PipelineRun, error) {
	return c.runs[parentName], nil
}

func withPipelineService(t *testing.T, svc *pkgpipeline.Service) {
	t.Helper()
	prev := pkgpipeline.ActiveService()
	pkgpipeline.SetActiveService(svc)
	t.Cleanup(func() { pkgpipeline.SetActiveService(prev) })
}

func newPipelineHandlerApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var te *types.Error
			if errors.As(err, &te) {
				switch te.Kind {
				case types.ErrInvalidArgument:
					code = fiber.StatusBadRequest
				case types.ErrNotFound:
					code = fiber.StatusNotFound
				case types.ErrUnavailable:
					code = fiber.StatusServiceUnavailable
				}
			}
			return c.Status(code).JSON(protocol.NewFailedResponse(err))
		},
	})
}

func mountPipelineHandlers(app *fiber.App) {
	app.Post("/service/automate/pipeline/apply", applyPipeline)
	app.Get("/service/automate/pipeline/list", listPipelines)
	app.Get("/service/automate/pipeline/get/:name", getPipeline)
	app.Get("/service/automate/pipeline/export/:name", exportPipeline)
	app.Delete("/service/automate/pipeline/delete/:name", deletePipeline)
	app.Post("/service/automate/pipeline/run", runPipeline)
	app.Get("/service/automate/pipeline/runs/:name", listPipelineRuns)
	app.Get("/service/automate/pipeline/runs", listPipelineRuns)
}

func samplePipelineYAML(name string) string {
	return `
name: ` + name + `
description: handler test
enabled: true
resumable: false
triggers:
  - type: event
    enabled: true
    event: demo.created
steps:
  - name: noop
    capability: core
    operation: echo
    params: {}
`
}

func TestPipelineHandlers(t *testing.T) {
	cat := newpipelineHandlerCatalog()
	completed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cat.runs["handler_pipe"] = []*model.PipelineRun{
		nil,
		{
			ID: 9, PipelineName: "handler_pipe", Status: 1, TriggerSource: "manual",
			EventID: "e1", EventType: "demo.created", CreatedAt: completed, StartedAt: completed,
			CompletedAt: &completed, Error: "boom",
		},
	}
	svc := pkgpipeline.NewService(cat)
	withPipelineService(t, svc)
	app := newPipelineHandlerApp()
	mountPipelineHandlers(app)

	yamlText := samplePipelineYAML("handler_pipe")
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantSubstr string
	}{
		{
			name:       "apply requires yaml",
			method:     http.MethodPost,
			path:       "/service/automate/pipeline/apply",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "yaml",
		},
		{
			name:       "apply accepts file_content fallback",
			method:     http.MethodPost,
			path:       "/service/automate/pipeline/apply",
			body:       `{"file_content":` + mustJSONString(t, yamlText) + `}`,
			wantStatus: http.StatusOK,
			wantSubstr: `"name":"handler_pipe"`,
		},
		{
			name:       "list includes applied pipeline",
			method:     http.MethodGet,
			path:       "/service/automate/pipeline/list",
			wantStatus: http.StatusOK,
			wantSubstr: "handler_pipe",
		},
		{
			name:       "get returns metadata",
			method:     http.MethodGet,
			path:       "/service/automate/pipeline/get/handler_pipe",
			wantStatus: http.StatusOK,
			wantSubstr: "handler_pipe",
		},
		{
			name:       "export returns yaml",
			method:     http.MethodGet,
			path:       "/service/automate/pipeline/export/handler_pipe",
			wantStatus: http.StatusOK,
			wantSubstr: "yaml",
		},
		{
			name:       "run requires name",
			method:     http.MethodPost,
			path:       "/service/automate/pipeline/run",
			body:       `{"event":{}}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "pipeline name is required",
		},
		{
			name:       "runs requires name",
			method:     http.MethodGet,
			path:       "/service/automate/pipeline/runs",
			wantStatus: http.StatusBadRequest,
			wantSubstr: "pipeline name is required",
		},
		{
			name:       "run without engine is unavailable",
			method:     http.MethodPost,
			path:       "/service/automate/pipeline/run",
			body:       `{"name":"handler_pipe"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantSubstr: "pipeline engine not ready",
		},
		{
			name:       "runs list skips nil and shapes fields",
			method:     http.MethodGet,
			path:       "/service/automate/pipeline/runs/handler_pipe",
			wantStatus: http.StatusOK,
			wantSubstr: `"error":"boom"`,
		},
		{
			name:       "delete removes definition",
			method:     http.MethodDelete,
			path:       "/service/automate/pipeline/delete/handler_pipe",
			wantStatus: http.StatusOK,
			wantSubstr: "deleted",
		},
	}

	prevEngine := pkgpipeline.ActiveEngine()
	pkgpipeline.SetActiveEngine(nil)
	t.Cleanup(func() { pkgpipeline.SetActiveEngine(prevEngine) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, http.NoBody)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode, string(raw))
			assert.Contains(t, string(raw), tt.wantSubstr)
		})
	}
}

func TestPipelineHandlersServiceUnavailable(t *testing.T) {
	withPipelineService(t, nil)
	app := newPipelineHandlerApp()
	mountPipelineHandlers(app)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "list", method: http.MethodGet, path: "/service/automate/pipeline/list"},
		{name: "get", method: http.MethodGet, path: "/service/automate/pipeline/get/x"},
		{name: "run", method: http.MethodPost, path: "/service/automate/pipeline/run", body: `{"name":"x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, http.NoBody)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	}
}

func mustJSONString(t *testing.T, s string) string {
	t.Helper()
	b, err := sonic.Marshal(s)
	require.NoError(t, err)
	return string(b)
}
