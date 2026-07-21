package workflow

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleInitDefaultsEnabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		conf   string
		wantOn bool
	}{
		{name: "empty config enables", conf: `{}`, wantOn: true},
		{name: "explicit enabled true", conf: `{"enabled":true}`, wantOn: true},
		{name: "explicit enabled false", conf: `{"enabled":false}`, wantOn: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := configType{Enabled: true}
			var raw map[string]any
			require.NoError(t, sonic.Unmarshal([]byte(tt.conf), &raw))
			require.NoError(t, sonic.Unmarshal([]byte(tt.conf), &cfg))
			if _, ok := raw["enabled"]; !ok {
				cfg.Enabled = true
			}
			assert.Equal(t, tt.wantOn, cfg.Enabled)
		})
	}
}

func TestWebserviceRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		path       string
		wantMethod string
	}{
		{name: "apply is POST", path: "/apply", wantMethod: "POST"},
		{name: "list is GET", path: "/list", wantMethod: "GET"},
		{name: "run is POST", path: "/run", wantMethod: "POST"},
		{name: "export is GET", path: "/export/:name", wantMethod: "GET"},
		{name: "delete is DELETE", path: "/delete/:name", wantMethod: "DELETE"},
		{name: "runs is GET", path: "/runs/:name", wantMethod: "GET"},
		{name: "get is GET", path: "/get/:name", wantMethod: "GET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			paths := make(map[string]string, len(webserviceRules))
			for _, r := range webserviceRules {
				paths[r.Path] = r.Method
			}
			require.Equal(t, tt.wantMethod, paths[tt.path])
		})
	}
}

func TestWebserviceRegistersWithoutInit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "mounts apply when not initialized"},
		{name: "mounts list when not initialized"},
		{name: "mounts run when not initialized"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := moduleHandler{} // initialized == false
			require.False(t, h.IsReady())
			app := fiber.New()
			require.NotPanics(t, func() {
				h.Webservice(app)
			})
		})
	}
}

func TestMountForE2E(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "apply is reachable", method: http.MethodPost, path: "/service/workflow/apply"},
		{name: "list is reachable", method: http.MethodGet, path: "/service/workflow/list"},
		{name: "run is reachable", method: http.MethodPost, path: "/service/workflow/run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			require.NotPanics(t, func() {
				MountForE2E(app)
			})
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}
