package web

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestWorkflowWebserviceRoutes(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ws := store.NewWorkflowStore(client)
	_, err := ws.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name:     "web-demo",
		Describe: "demo workflow",
		Enabled:  true,
		Inputs: []types.WorkflowInputDef{
			{Name: "url", Type: types.WorkflowInputTypeString, Required: true, Description: "Target URL"},
			{Name: "count", Type: types.WorkflowInputTypeNumber, Default: 1},
		},
		Pipeline: []string{"step1"},
		Tasks: []types.WorkflowTask{
			{ID: "step1", Action: "mapper:", Describe: "echo step", Params: types.KV{"msg": "{{input.url}}"}},
		},
		Triggers: []types.WorkflowTriggerDef{
			{Type: "manual", Enabled: true},
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		method       string
		path         string
		body         io.Reader
		contentType  string
		withCookie   bool
		wantStatus   int
		wantContains []string
		wantHeader   map[string]string
	}{
		{
			name:       "list unauthenticated redirects",
			method:     http.MethodGet,
			path:       "/service/web/workflows",
			withCookie: false,
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "list page shows workflow",
			method:     http.MethodGet,
			path:       "/service/web/workflows",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Workflows",
				"web-demo",
				`data-testid="workflow-table"`,
				`data-testid="btn-disable-workflow-web-demo"`,
			},
		},
		{
			name:       "list partial",
			method:     http.MethodGet,
			path:       "/service/web/workflows/list",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="workflow-row-web-demo"`,
				`data-testid="btn-disable-workflow-web-demo"`,
			},
		},
		{
			name:       "detail page",
			method:     http.MethodGet,
			path:       "/service/web/workflows/web-demo",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"demo workflow",
				`data-testid="workflow-run-form"`,
				`data-testid="workflow-dag"`,
				`data-testid="workflow-detail-tabs"`,
				`data-testid="workflow-tab-yaml"`,
				`data-testid="workflow-yaml"`,
				"mapper:",
				"manual",
				`data-testid="workflow-triggers-table"`,
			},
		},
		{
			name:       "runs page",
			method:     http.MethodGet,
			path:       "/service/web/workflows/web-demo/runs",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Run History",
				`data-testid="workflow-runs-empty"`,
			},
		},
		{
			name:       "runs list partial",
			method:     http.MethodGet,
			path:       "/service/web/workflows/web-demo/runs/list",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="workflow-runs-empty"`,
			},
		},
		{
			name:         "run steps for missing run",
			method:       http.MethodGet,
			path:         "/service/web/workflows/web-demo/runs/999999/steps",
			withCookie:   true,
			wantStatus:   http.StatusNotFound,
			wantContains: []string{"run not found"},
		},
		{
			name:        "run now missing required input",
			method:      http.MethodPost,
			path:        "/service/web/workflows/web-demo/run",
			body:        strings.NewReader(url.Values{}.Encode()),
			contentType: "application/x-www-form-urlencoded",
			withCookie:  true,
			wantStatus:  http.StatusUnprocessableEntity,
			wantContains: []string{
				"required input",
			},
		},
		{
			name:   "run now starts async",
			method: http.MethodPost,
			path:   "/service/web/workflows/web-demo/run",
			body: strings.NewReader(url.Values{
				"url": {"https://example.com"},
			}.Encode()),
			contentType: "application/x-www-form-urlencoded",
			withCookie:  true,
			wantStatus:  http.StatusOK,
			wantHeader: map[string]string{
				"HX-Redirect": "/service/web/workflows/web-demo/runs",
			},
		},
		{
			name:        "run now json body",
			method:      http.MethodPost,
			path:        "/service/web/workflows/web-demo/run",
			body:        strings.NewReader(`{"url":"https://json.example"}`),
			contentType: "application/json",
			withCookie:  true,
			wantStatus:  http.StatusOK,
			wantHeader: map[string]string{
				"HX-Redirect": "/service/web/workflows/web-demo/runs",
			},
		},
		{
			name:       "detail not found",
			method:     http.MethodGet,
			path:       "/service/web/workflows/missing-wf",
			withCookie: true,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader = http.NoBody
			if tt.body != nil {
				body = tt.body
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.withCookie {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
				AttachCSRFForTest(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			for k, v := range tt.wantHeader {
				assert.Equal(t, v, resp.Header.Get(k))
			}
			if len(tt.wantContains) == 0 {
				return
			}
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			html := string(raw)
			for _, sub := range tt.wantContains {
				assert.Contains(t, html, sub)
			}
		})
	}
}

func TestSetWorkflowEnabledDisableAndEnable(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ws := store.NewWorkflowStore(client)
	_, err := ws.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name:     "toggle-wf",
		Enabled:  true,
		Pipeline: []string{"step1"},
		Tasks:    []types.WorkflowTask{{ID: "step1", Action: "mapper:"}},
		Triggers: []types.WorkflowTriggerDef{{Type: "manual", Enabled: true}},
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		enabled      bool
		wantContains []string
		wantEnabled  bool
	}{
		{
			name:         "disable workflow",
			enabled:      false,
			wantContains: []string{`data-testid="btn-enable-workflow-toggle-wf"`},
			wantEnabled:  false,
		},
		{
			name:         "enable workflow",
			enabled:      true,
			wantContains: []string{`data-testid="btn-disable-workflow-toggle-wf"`},
			wantEnabled:  true,
		},
		{
			name:        "missing workflow returns not found",
			enabled:     false,
			wantEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/service/web/workflows/toggle-wf/enabled"
			if tt.name == "missing workflow returns not found" {
				path = "/service/web/workflows/missing-toggle/enabled"
			}
			body := bytes.NewBufferString(`{"enabled":` + boolJSON(tt.enabled) + `}`)
			req := httptest.NewRequest(http.MethodPut, path, body)
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			if tt.name == "missing workflow returns not found" {
				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
				return
			}
			require.Equal(t, http.StatusOK, resp.StatusCode)
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			html := string(raw)
			for _, sub := range tt.wantContains {
				assert.Contains(t, html, sub)
			}
			row, err := ws.GetDefinitionByName(ctx, "toggle-wf")
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, row.Workflow.Enabled)
		})
	}
}

func TestSetWorkflowTriggerEnabledDisableAndEnable(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ws := store.NewWorkflowStore(client)
	_, err := ws.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name:     "toggle-trigger-wf",
		Enabled:  true,
		Pipeline: []string{"step1"},
		Tasks:    []types.WorkflowTask{{ID: "step1", Action: "mapper:"}},
		Triggers: []types.WorkflowTriggerDef{
			{Type: "manual", Enabled: true},
			{Type: "cron", Enabled: true, Rule: types.KV{"cron": "@hourly"}},
		},
	})
	require.NoError(t, err)
	dto, err := ws.GetDefinitionByName(ctx, "toggle-trigger-wf")
	require.NoError(t, err)
	require.Len(t, dto.Triggers, 2)
	cronID := dto.Triggers[1].ID

	tests := []struct {
		name         string
		enabled      bool
		wantContains []string
		wantEnabled  bool
	}{
		{
			name:         "disable trigger",
			enabled:      false,
			wantContains: []string{`data-testid="btn-enable-trigger-` + strconv.FormatInt(cronID, 10) + `"`},
			wantEnabled:  false,
		},
		{
			name:         "enable trigger",
			enabled:      true,
			wantContains: []string{`data-testid="btn-disable-trigger-` + strconv.FormatInt(cronID, 10) + `"`},
			wantEnabled:  true,
		},
		{
			name:        "foreign trigger returns not found",
			enabled:     false,
			wantEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := cronID
			if tt.name == "foreign trigger returns not found" {
				id = 999999
			}
			body := bytes.NewBufferString(`{"enabled":` + boolJSON(tt.enabled) + `}`)
			req := httptest.NewRequest(http.MethodPut,
				"/service/web/workflows/toggle-trigger-wf/triggers/"+strconv.FormatInt(id, 10)+"/enabled", body)
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			if tt.name == "foreign trigger returns not found" {
				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
				return
			}
			require.Equal(t, http.StatusOK, resp.StatusCode)
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			html := string(raw)
			for _, sub := range tt.wantContains {
				assert.Contains(t, html, sub)
			}
			updated, err := ws.GetDefinitionByName(ctx, "toggle-trigger-wf")
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, updated.Triggers[1].Enabled)
			assert.True(t, updated.Triggers[0].Enabled)
		})
	}
}

func TestWorkflowRunSteps(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ws := store.NewWorkflowStore(client)
	rs := store.NewWorkflowRunStore(client)
	_, err := ws.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name:     "steps-wf",
		Enabled:  true,
		Pipeline: []string{"step1"},
		Tasks:    []types.WorkflowTask{{ID: "step1", Action: "mapper:"}},
		Triggers: []types.WorkflowTriggerDef{{Type: "manual", Enabled: true}},
	})
	require.NoError(t, err)
	dto, err := ws.GetDefinitionByName(ctx, "steps-wf")
	require.NoError(t, err)
	run, err := rs.CreateRun(ctx, dto.Workflow.ID, "steps-wf", "db", "manual", nil, nil)
	require.NoError(t, err)
	_, err = rs.CreateStepRun(ctx, run.ID, "step1", "step1", "mapper:", "mapper", map[string]any{"msg": "hi"}, 1)
	require.NoError(t, err)

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantContains []string
	}{
		{
			name:       "returns task runs detail",
			path:       "/service/web/workflows/steps-wf/runs/" + strconv.FormatInt(run.ID, 10) + "/steps",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="workflow-step-runs-detail"`,
				`data-testid="workflow-step-row-step1"`,
				"mapper:",
				"Input",
			},
		},
		{
			name:         "rejects run from other workflow name",
			path:         "/service/web/workflows/other-wf/runs/" + strconv.FormatInt(run.ID, 10) + "/steps",
			wantStatus:   http.StatusNotFound,
			wantContains: []string{"run not found"},
		},
		{
			name:         "invalid run id",
			path:         "/service/web/workflows/steps-wf/runs/abc/steps",
			wantStatus:   http.StatusBadRequest,
			wantContains: []string{"invalid run ID"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if len(tt.wantContains) == 0 {
				return
			}
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			html := string(raw)
			for _, sub := range tt.wantContains {
				assert.Contains(t, html, sub)
			}
		})
	}
}

func boolJSON(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func TestParseWorkflowRunInputs(t *testing.T) {
	t.Parallel()
	declared := []types.WorkflowInputDef{
		{Name: "url", Type: types.WorkflowInputTypeString, Required: true},
		{Name: "count", Type: types.WorkflowInputTypeNumber},
		{Name: "flag", Type: types.WorkflowInputTypeBoolean},
		{Name: "meta", Type: types.WorkflowInputTypeJSON},
	}
	tests := []struct {
		name    string
		body    string
		want    types.KV
		wantErr bool
	}{
		{
			name: "nested inputs object",
			body: `{"inputs":{"url":"https://a","count":2,"flag":true,"meta":{"k":1}}}`,
			want: types.KV{"url": "https://a", "count": float64(2), "flag": true, "meta": map[string]any{"k": float64(1)}},
		},
		{
			name: "flat object",
			body: `{"url":"u"}`,
			want: types.KV{"url": "u"},
		},
		{
			name:    "invalid json",
			body:    `{`,
			wantErr: true,
		},
		{
			name: "empty body",
			body: "",
			want: types.KV{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseWorkflowRunJSONBody([]byte(tt.body), declared)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCoerceWorkflowInputString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		def     types.WorkflowInputDef
		raw     string
		want    any
		wantErr bool
	}{
		{name: "string", def: types.WorkflowInputDef{Name: "a", Type: types.WorkflowInputTypeString}, raw: "hi", want: "hi"},
		{name: "number", def: types.WorkflowInputDef{Name: "n", Type: types.WorkflowInputTypeNumber}, raw: "3.5", want: 3.5},
		{name: "bool true", def: types.WorkflowInputDef{Name: "b", Type: types.WorkflowInputTypeBoolean}, raw: "true", want: true},
		{name: "bool false", def: types.WorkflowInputDef{Name: "b", Type: types.WorkflowInputTypeBoolean}, raw: "false", want: false},
		{name: "json object", def: types.WorkflowInputDef{Name: "j", Type: types.WorkflowInputTypeJSON}, raw: `{"a":1}`, want: map[string]any{"a": float64(1)}},
		{name: "bad number", def: types.WorkflowInputDef{Name: "n", Type: types.WorkflowInputTypeNumber}, raw: "x", wantErr: true},
		{name: "bad json scalar", def: types.WorkflowInputDef{Name: "j", Type: types.WorkflowInputTypeJSON}, raw: `"x"`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := coerceWorkflowInputString(tt.def, tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorkflowWebserviceRulesRegistered(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
	}{
		{name: "list", path: "/workflows"},
		{name: "list partial", path: "/workflows/list"},
		{name: "detail", path: "/workflows/:name"},
		{name: "runs", path: "/workflows/:name/runs"},
		{name: "runs list", path: "/workflows/:name/runs/list"},
		{name: "run", path: "/workflows/:name/run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found := false
			for _, r := range workflowWebserviceRules {
				if r.Path == tt.path {
					found = true
					break
				}
			}
			assert.True(t, found, "missing path %s", tt.path)
		})
	}
}
