package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/capability"
	capfunctions "github.com/flowline-io/flowbot/pkg/capability/functions"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type webFnExecProvider struct {
	stdout string
}

func (p *webFnExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{
		Env: &webFnScriptedEnv{stdout: p.stdout},
	}, nil
}

type webFnScriptedEnv struct {
	env.OSExecutionEnv
	stdout string
}

func (e *webFnScriptedEnv) Exec(_ context.Context, _ env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	return result.Ok[env.Capture, result.ExecutionError](env.Capture{
		Stdout:   e.stdout,
		ExitCode: 0,
	})
}

func wireFunctionServiceForTest(t *testing.T, client *store.Client, stdout string) *pkgfunctions.Service {
	t.Helper()
	fs := store.NewFunctionStore(client)
	catalog := store.NewFunctionCatalogAdapter(fs)
	svc := pkgfunctions.NewService(catalog, &webFnExecProvider{stdout: stdout})
	svc.SetChecker(dcg.AllowAllChecker{})
	pkgfunctions.SetActiveService(svc)
	require.NoError(t, capfunctions.Register())
	t.Cleanup(func() {
		pkgfunctions.SetActiveService(nil)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpInvoke)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpGet)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpHealth)
		hub.Default.Unregister(hub.CapFunctions)
	})
	return svc
}

func TestFunctionWebCreateDraftPublishTry(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })
	wireFunctionServiceForTest(t, client, `{"ok":true}`)

	form := url.Values{}
	form.Set("name", "web-fn")
	form.Set("entrypoint", "main.py")
	req := httptest.NewRequest(http.MethodPost, "/service/web/functions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/service/web/functions/web-fn", resp.Header.Get("HX-Redirect"))

	listReq := httptest.NewRequest(http.MethodGet, "/service/web/functions", http.NoBody)
	addWebAuth(listReq)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody, err := io.ReadAll(listResp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(listBody), "web-fn")
	assert.Contains(t, string(listBody), "Draft")
	assert.Contains(t, string(listBody), "/static/js/function-stats.js")
	assert.Contains(t, string(listBody), `hx-get="/service/web/functions/stats?days=30&amp;groupBy=day"`)
	assert.Contains(t, string(listBody), `hx-get="/service/web/functions/list"`)

	statsReq := httptest.NewRequest(http.MethodGet, "/service/web/functions/stats?days=30&groupBy=day", http.NoBody)
	addWebAuth(statsReq)
	statsResp, err := app.Test(statsReq)
	require.NoError(t, err)
	defer statsResp.Body.Close()
	require.Equal(t, http.StatusOK, statsResp.StatusCode)
	statsBody, err := io.ReadAll(statsResp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(statsBody), `data-testid="function-stats-container"`)
	assert.Contains(t, string(statsBody), `data-testid="function-stats-summary"`)
	assert.Contains(t, string(statsBody), "Total Functions")
	assert.Contains(t, string(statsBody), "Success Rate Trend")
	assert.Contains(t, string(statsBody), "Published Versions")

	namedStatsReq := httptest.NewRequest(http.MethodGet, "/service/web/functions/web-fn/stats?days=30&groupBy=day", http.NoBody)
	addWebAuth(namedStatsReq)
	namedStatsResp, err := app.Test(namedStatsReq)
	require.NoError(t, err)
	defer namedStatsResp.Body.Close()
	require.Equal(t, http.StatusOK, namedStatsResp.StatusCode)
	namedStatsBody, err := io.ReadAll(namedStatsResp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(namedStatsBody), `data-testid="function-stats-container"`)
	assert.NotContains(t, string(namedStatsBody), `data-testid="function-stats-summary"`)

	editReq := httptest.NewRequest(http.MethodGet, "/service/web/functions/web-fn", http.NoBody)
	addWebAuth(editReq)
	editResp, err := app.Test(editReq)
	require.NoError(t, err)
	defer editResp.Body.Close()
	require.Equal(t, http.StatusOK, editResp.StatusCode)
	editBody, err := io.ReadAll(editResp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(editBody), "function-editor")
	assert.Contains(t, string(editBody), "function-call-links")
	assert.Contains(t, string(editBody), "/service/functions/call/web-fn")
	assert.Contains(t, string(editBody), `data-testid="function-call-url"`)
	assert.NotContains(t, string(editBody), `x-text="callURL"`)
	assert.Contains(t, string(editBody), "/static/js/function-stats.js")
	assert.Contains(t, string(editBody), `data-testid="tab-stats"`)
	assert.Contains(t, string(editBody), `/service/web/functions/web-fn/stats?days=30&amp;groupBy=day`)
	assert.Contains(t, string(editBody), pkgconfig.MaskedSecret)
	assert.NotContains(t, string(editBody), "secret-token")

	svc := pkgfunctions.ActiveService()
	draft, err := svc.GetDraft(context.Background(), "web-fn")
	require.NoError(t, err)

	meta := "name: web-fn\nhttp:\n  auth:\n    token: " + pkgconfig.MaskedSecret + "\nenv:\n  mode: web\n"
	saveBody, err := sonic.MarshalString(map[string]any{
		"metadata":   meta,
		"entrypoint": "main.py",
		"source":     "print('{\"ok\":true}')\n",
		"version":    draft.Version,
	})
	require.NoError(t, err)
	saveReq := httptest.NewRequest(http.MethodPut, "/service/web/functions/web-fn", strings.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	addWebAuth(saveReq)
	saveResp, err := app.Test(saveReq)
	require.NoError(t, err)
	defer saveResp.Body.Close()
	require.Equal(t, http.StatusOK, saveResp.StatusCode)
	var saved struct {
		Version int `json:"version"`
	}
	require.NoError(t, sonic.ConfigDefault.NewDecoder(saveResp.Body).Decode(&saved))
	assert.Equal(t, draft.Version+1, saved.Version)

	conflictBody, err := sonic.MarshalString(map[string]any{
		"metadata":   meta,
		"entrypoint": "main.py",
		"source":     "print(1)\n",
		"version":    draft.Version,
	})
	require.NoError(t, err)
	conflictReq := httptest.NewRequest(http.MethodPut, "/service/web/functions/web-fn", strings.NewReader(conflictBody))
	conflictReq.Header.Set("Content-Type", "application/json")
	addWebAuth(conflictReq)
	conflictResp, err := app.Test(conflictReq)
	require.NoError(t, err)
	defer conflictResp.Body.Close()
	require.Equal(t, http.StatusConflict, conflictResp.StatusCode)

	pubBody, err := sonic.MarshalString(map[string]any{"version": saved.Version})
	require.NoError(t, err)
	pubReq := httptest.NewRequest(http.MethodPost, "/service/web/functions/web-fn/publish", strings.NewReader(pubBody))
	pubReq.Header.Set("Content-Type", "application/json")
	addWebAuth(pubReq)
	pubResp, err := app.Test(pubReq)
	require.NoError(t, err)
	defer pubResp.Body.Close()
	require.Equal(t, http.StatusOK, pubResp.StatusCode)
	var published map[string]any
	require.NoError(t, sonic.ConfigDefault.NewDecoder(pubResp.Body).Decode(&published))
	require.NotNil(t, published["version"])

	tryBody, err := sonic.MarshalString(map[string]any{"event": map[string]any{"x": 1}})
	require.NoError(t, err)
	tryReq := httptest.NewRequest(http.MethodPost, "/service/web/functions/web-fn/try", strings.NewReader(tryBody))
	tryReq.Header.Set("Content-Type", "application/json")
	addWebAuth(tryReq)
	tryResp, err := app.Test(tryReq)
	require.NoError(t, err)
	defer tryResp.Body.Close()
	require.Equal(t, http.StatusOK, tryResp.StatusCode)
	raw, err := io.ReadAll(tryResp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"ok"`)
}

func TestFunctionWebTrySandboxUnavailable(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })
	fs := store.NewFunctionStore(client)
	catalog := store.NewFunctionCatalogAdapter(fs)
	svc := pkgfunctions.NewService(catalog, &dockerDownWebExecProvider{})
	svc.SetChecker(dcg.AllowAllChecker{})
	pkgfunctions.SetActiveService(svc)
	require.NoError(t, capfunctions.Register())
	t.Cleanup(func() {
		pkgfunctions.SetActiveService(nil)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpInvoke)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpGet)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpHealth)
		hub.Default.Unregister(hub.CapFunctions)
	})

	form := url.Values{}
	form.Set("name", "no-docker-fn")
	form.Set("entrypoint", "main.py")
	createReq := httptest.NewRequest(http.MethodPost, "/service/web/functions", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addWebAuth(createReq)
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusOK, createResp.StatusCode)

	draft, err := svc.GetDraft(context.Background(), "no-docker-fn")
	require.NoError(t, err)
	pubBody, err := sonic.MarshalString(map[string]any{"version": draft.Version})
	require.NoError(t, err)
	pubReq := httptest.NewRequest(http.MethodPost, "/service/web/functions/no-docker-fn/publish", strings.NewReader(pubBody))
	pubReq.Header.Set("Content-Type", "application/json")
	addWebAuth(pubReq)
	pubResp, err := app.Test(pubReq)
	require.NoError(t, err)
	defer pubResp.Body.Close()
	require.Equal(t, http.StatusOK, pubResp.StatusCode)

	tryBody, err := sonic.MarshalString(map[string]any{"event": map[string]any{}})
	require.NoError(t, err)
	tryReq := httptest.NewRequest(http.MethodPost, "/service/web/functions/no-docker-fn/try", strings.NewReader(tryBody))
	tryReq.Header.Set("Content-Type", "application/json")
	addWebAuth(tryReq)
	tryResp, err := app.Test(tryReq)
	require.NoError(t, err)
	defer tryResp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, tryResp.StatusCode)
	raw, err := io.ReadAll(tryResp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "UNAVAILABLE")
	assert.Contains(t, string(raw), "Docker is not running")
	assert.NotContains(t, string(raw), "docker.sock")
}

type dockerDownWebExecProvider struct{}

func (dockerDownWebExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{Env: dockerDownWebEnv{}}, nil
}

type dockerDownWebEnv struct {
	env.OSExecutionEnv
}

func (dockerDownWebEnv) Exec(_ context.Context, _ env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	cause := errors.New("failed to connect to the docker API at unix:///var/run/docker.sock")
	return result.Err[env.Capture, result.ExecutionError](result.NewExecutionError("spawn_error", cause.Error(), cause))
}

func TestFunctionWebserviceRulesRegistered(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
	}{
		{name: "list", path: "/functions"},
		{name: "list partial", path: "/functions/list"},
		{name: "stats", path: "/functions/stats"},
		{name: "detail", path: "/functions/:name"},
		{name: "detail stats", path: "/functions/:name/stats"},
		{name: "runs", path: "/functions/:name/runs"},
		{name: "try", path: "/functions/:name/try"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found := false
			for _, r := range functionWebserviceRules {
				if r.Path == tt.path {
					found = true
					break
				}
			}
			assert.True(t, found, "expected rule path %s", tt.path)
		})
	}
}
