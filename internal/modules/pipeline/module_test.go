package pipeline

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetPipelineModuleState(t *testing.T) {
	t.Helper()
	handler = moduleHandler{}
	config = configType{}
}

func TestModuleInitAndRoutes(t *testing.T) {
	resetPipelineModuleState(t)
	t.Cleanup(func() { resetPipelineModuleState(t) })

	require.NoError(t, InitForE2E(json.RawMessage(`{"enabled":true}`)))
	assert.True(t, handler.IsReady())

	app := fiber.New()
	MountForE2E(app)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "apply is reachable", method: http.MethodPost, path: "/service/pipeline/apply"},
		{name: "list is reachable", method: http.MethodGet, path: "/service/pipeline/list"},
		{name: "run is reachable", method: http.MethodPost, path: "/service/pipeline/run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(tt.method, tt.path, http.NoBody)
			require.NoError(t, err)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}
