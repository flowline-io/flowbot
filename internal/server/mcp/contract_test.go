package mcp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

func TestContractAnnotations(t *testing.T) {
	t.Parallel()
	descs := []hub.Descriptor{
		{Type: hub.CapKarakeep, Operations: []hub.Operation{
			{Name: "create", Description: "Create a bookmark"},
			{Name: "delete", Description: "Delete a bookmark"},
			{Name: "list", Description: "List bookmarks"},
			{Name: "update", Description: "Update a bookmark"},
		}},
		{Type: hub.CapExample, Operations: []hub.Operation{
			{Name: "list", Description: "List items"},
		}},
		{Type: hub.CapTransmission, Operations: []hub.Operation{
			{Name: "stop", Description: "Stop torrents by ID"},
			{Name: "remove", Description: "Remove torrents by ID"},
		}},
	}
	byName := map[string]ToolSpec{}
	for _, spec := range BuildCatalog(descs, nil, nil) {
		byName[spec.Name] = spec
		ann := toolAnnotations(spec)
		assert.Equal(t, spec.Group+": "+spec.Description, ann.Title)
		require.NotNil(t, ann.OpenWorldHint)
	}

	create := toolAnnotations(byName["karakeep.create"])
	assert.False(t, create.ReadOnlyHint)
	require.NotNil(t, create.DestructiveHint)
	assert.False(t, *create.DestructiveHint)
	assert.False(t, create.IdempotentHint)
	assert.True(t, *create.OpenWorldHint)
	assert.Nil(t, byName["karakeep.create"].Output)

	del := toolAnnotations(byName["karakeep.delete"])
	require.NotNil(t, del.DestructiveHint)
	assert.True(t, *del.DestructiveHint)

	remove := toolAnnotations(byName["transmission.remove"])
	require.NotNil(t, remove.DestructiveHint)
	assert.True(t, *remove.DestructiveHint)

	stop := toolAnnotations(byName["transmission.stop"])
	assert.False(t, stop.ReadOnlyHint)
	require.NotNil(t, stop.DestructiveHint)
	assert.False(t, *stop.DestructiveHint)
	assert.False(t, capability.IsMutation("stop"))

	update := toolAnnotations(byName["karakeep.update"])
	assert.True(t, update.IdempotentHint)
	assert.False(t, *update.DestructiveHint)

	list := toolAnnotations(byName["karakeep.list"])
	assert.True(t, list.ReadOnlyHint)
	assert.Nil(t, list.DestructiveHint)
	require.NotNil(t, byName["karakeep.list"].Output)

	example := toolAnnotations(byName["example.list"])
	assert.False(t, *example.OpenWorldHint)

	pipeline := toolAnnotations(byName["pipeline.run"])
	assert.False(t, pipeline.ReadOnlyHint)
	assert.False(t, *pipeline.DestructiveHint)
	assert.False(t, *pipeline.OpenWorldHint)
	require.NotNil(t, byName["pipeline.list"].Output)
	require.NotNil(t, byName["hub.apps"].Output)
	require.NotNil(t, byName["hub.health"].Output)
	assert.False(t, *toolAnnotations(byName["hub.health"]).OpenWorldHint)
}

func TestContextForToolCopiesRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Post("/probe", func(c fiber.Ctx) error {
		return adaptor.HTTPHandlerWithContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := contextForTool(c, r, Identity{})
			w.Header().Set("X-Tool-Request-ID", requestIDFrom(ctx))
			w.WriteHeader(http.StatusNoContent)
		}))(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/probe", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
	assert.Equal(t, resp.Header.Get("X-Request-ID"), resp.Header.Get("X-Tool-Request-ID"))
}

func TestToolOKAddsTrace(t *testing.T) {
	t.Parallel()
	ctx := withRequestID(t.Context(), "req-test")
	res, err := toolOK(ctx, []string{"app"}, true)
	require.NoError(t, err)
	require.False(t, res.IsError)
	body, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, body["items"])
	trace, ok := body["trace"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "req-test", trace["request_id"])
	assert.NotContains(t, trace, "correlation_id")
	require.NoError(t, pkgmcp.ValidateOutput(pkgmcp.OutputSchema("hub", "apps"), body))
}

func TestGetRunListIsRunsObject(t *testing.T) {
	t.Parallel()
	got, err := filterModelRuns([]struct{ ID int64 }{{ID: 1}}, nil)
	require.NoError(t, err)
	runs, ok := got.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, runs, "runs")

	ctx := withRequestID(t.Context(), "req-test")
	res, err := toolOK(ctx, got, false)
	require.NoError(t, err)
	body, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, body, "runs")
	assert.NotContains(t, body, "items")
	trace, ok := body["trace"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "req-test", trace["request_id"])
	require.NoError(t, pkgmcp.ValidateOutput(pkgmcp.OutputSchema("pipeline", "get_run"), body))
}
