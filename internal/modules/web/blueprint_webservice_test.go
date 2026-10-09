package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/pipeline"
)

func wireBlueprintService(t *testing.T, client *store.Client) {
	t.Helper()
	ps := store.NewPipelineStore(client)
	adapter := store.PipelineCatalogAdapter{S: ps}
	pipeline.SetActiveBlueprintService(pipeline.NewBlueprintService(ps, adapter))
	t.Cleanup(func() { pipeline.SetActiveBlueprintService(nil) })
}

func TestBlueprintListPage(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })
	wireBlueprintService(t, client)

	req := httptest.NewRequest(http.MethodGet, "/service/web/blueprints", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)
	assert.Contains(t, html, "webhook_notify")
	assert.Contains(t, html, `data-testid="blueprint-table"`)
}

func TestBlueprintDownloadOfficial(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })
	wireBlueprintService(t, client)

	req := httptest.NewRequest(http.MethodGet, "/service/web/blueprints/builtin/webhook_notify/download", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "kind: pipeline_blueprint")
	assert.Contains(t, string(body), "id: webhook_notify")
}
