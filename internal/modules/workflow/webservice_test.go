package workflow

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	pkgworkflow "github.com/flowline-io/flowbot/pkg/workflow"
)

type handlerCatalog struct {
	meta map[string]*types.WorkflowMetadata
	defs []*gen.Workflow
}

func (c *handlerCatalog) GetMetadata(_ context.Context, name string) (*types.WorkflowMetadata, error) {
	meta, ok := c.meta[name]
	if !ok {
		return nil, types.Errorf(types.ErrNotFound, "workflow %s", name)
	}
	return meta, nil
}

func (c *handlerCatalog) ApplyDefinition(_ context.Context, meta *types.WorkflowMetadata) (*gen.Workflow, error) {
	if meta == nil || meta.Name == "" {
		return nil, errors.New("invalid meta")
	}
	row := &gen.Workflow{ID: 7, Name: meta.Name, Enabled: meta.Enabled, Describe: meta.Describe}
	c.meta[meta.Name] = meta
	c.defs = append(c.defs, row)
	return row, nil
}

func (c *handlerCatalog) ListDefinitions(context.Context) ([]*gen.Workflow, error) {
	return c.defs, nil
}

func (c *handlerCatalog) DeleteDefinitionByName(_ context.Context, name string) error {
	delete(c.meta, name)
	out := c.defs[:0]
	for _, d := range c.defs {
		if d != nil && d.Name != name {
			out = append(out, d)
		}
	}
	c.defs = out
	return nil
}

func (*handlerCatalog) ListRunsByName(context.Context, string) ([]*gen.WorkflowRun, error) {
	return nil, nil
}

type handlerRunStore struct {
	mu     sync.Mutex
	nextID int64
	runs   []*gen.WorkflowRun
}

func (s *handlerRunStore) CreateRun(_ context.Context, workflowID int64, workflowName, workflowFile, triggerType string, _, inputParams map[string]any) (*gen.WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	var wfID *int64
	if workflowID != 0 {
		wfID = &workflowID
	}
	run := &gen.WorkflowRun{
		ID:           s.nextID,
		WorkflowID:   wfID,
		WorkflowName: workflowName,
		WorkflowFile: workflowFile,
		TriggerType:  triggerType,
		InputParams:  inputParams,
	}
	s.runs = append(s.runs, run)
	return run, nil
}

func (*handlerRunStore) UpdateRunStatus(context.Context, int64, int, string) error { return nil }
func (*handlerRunStore) CreateStepRun(context.Context, int64, string, string, string, string, map[string]any, int) (*gen.WorkflowStepRun, error) {
	return nil, nil
}
func (*handlerRunStore) UpdateStepRun(context.Context, int64, int, map[string]any, string, int) error {
	return nil
}
func (*handlerRunStore) SaveCheckpoint(context.Context, int64, any) error { return nil }
func (*handlerRunStore) GetIncompleteRuns(context.Context) ([]*gen.WorkflowRun, error) {
	return nil, nil
}
func (*handlerRunStore) GetCheckpoint(context.Context, int64, any) error { return nil }
func (*handlerRunStore) GetRun(context.Context, int64) (*gen.WorkflowRun, error) {
	return nil, types.Errorf(types.ErrNotFound, "run")
}
func (*handlerRunStore) UpdateRunHeartbeat(context.Context, int64) error { return nil }

func withWorkflowService(t *testing.T, svc *pkgworkflow.Service) {
	t.Helper()
	prev := pkgworkflow.ActiveService()
	pkgworkflow.SetReloadService(svc)
	t.Cleanup(func() {
		pkgworkflow.SetReloadService(prev)
	})
}

func newWorkflowHandlerApp() *fiber.App {
	app := fiber.New(fiber.Config{
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
	// Mount handlers without Authorize so unit tests cover request/response mapping.
	app.Post("/service/workflow/apply", applyWorkflow)
	app.Get("/service/workflow/list", listWorkflows)
	app.Get("/service/workflow/get/:name", getWorkflow)
	app.Get("/service/workflow/export/:name", exportWorkflow)
	app.Delete("/service/workflow/delete/:name", deleteWorkflow)
	app.Post("/service/workflow/run", runWorkflow)
	app.Get("/service/workflow/runs/:name", listWorkflowRuns)
	return app
}

func TestWorkflowHandlers(t *testing.T) {
	echoYAML := `
name: handler-echo
describe: handler test
enabled: true
pipeline:
  - build
tasks:
  - id: build
    action: "mapper:"
    params:
      message: hello
`
	catalog := &handlerCatalog{
		meta: map[string]*types.WorkflowMetadata{},
	}
	runs := &handlerRunStore{}
	svc := pkgworkflow.NewService(catalog, runs, nil, nil)
	withWorkflowService(t, svc)
	app := newWorkflowHandlerApp()

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
			path:       "/service/workflow/apply",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "yaml",
		},
		{
			name:       "apply stores definition",
			method:     http.MethodPost,
			path:       "/service/workflow/apply",
			body:       `{"yaml":` + mustJSONString(t, echoYAML) + `}`,
			wantStatus: http.StatusOK,
			wantSubstr: `"name":"handler-echo"`,
		},
		{
			name:       "list includes applied workflow",
			method:     http.MethodGet,
			path:       "/service/workflow/list",
			wantStatus: http.StatusOK,
			wantSubstr: "handler-echo",
		},
		{
			name:       "get returns metadata",
			method:     http.MethodGet,
			path:       "/service/workflow/get/handler-echo",
			wantStatus: http.StatusOK,
			wantSubstr: "handler-echo",
		},
		{
			name:       "export returns yaml",
			method:     http.MethodGet,
			path:       "/service/workflow/export/handler-echo",
			wantStatus: http.StatusOK,
			wantSubstr: "yaml",
		},
		{
			name:       "run requires name",
			method:     http.MethodPost,
			path:       "/service/workflow/run",
			body:       `{"input":{}}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "name",
		},
		{
			name:       "run accepted",
			method:     http.MethodPost,
			path:       "/service/workflow/run",
			body:       `{"name":"handler-echo","input":{}}`,
			wantStatus: http.StatusAccepted,
			wantSubstr: "run_id",
		},
		{
			name:       "runs list ok",
			method:     http.MethodGet,
			path:       "/service/workflow/runs/handler-echo",
			wantStatus: http.StatusOK,
			wantSubstr: "runs",
		},
		{
			name:       "delete removes definition",
			method:     http.MethodDelete,
			path:       "/service/workflow/delete/handler-echo",
			wantStatus: http.StatusOK,
			wantSubstr: "deleted",
		},
		{
			name:       "get missing after delete",
			method:     http.MethodGet,
			path:       "/service/workflow/get/handler-echo",
			wantStatus: http.StatusNotFound,
			wantSubstr: "handler-echo",
		},
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
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode, string(raw))
			assert.Contains(t, string(raw), tt.wantSubstr)
		})
	}
}

func TestWorkflowHandlersServiceUnavailable(t *testing.T) {
	withWorkflowService(t, nil)
	app := newWorkflowHandlerApp()
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "list", method: http.MethodGet, path: "/service/workflow/list"},
		{name: "get", method: http.MethodGet, path: "/service/workflow/get/x"},
		{name: "run", method: http.MethodPost, path: "/service/workflow/run", body: `{"name":"x"}`},
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
