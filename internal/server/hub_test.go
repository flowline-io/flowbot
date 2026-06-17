package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func setupHubTestApp() *fiber.App {
	app := newTestApp()
	ctl := &Controller{}
	app.Get("/hub/apps", ctl.hubApps)
	app.Get("/hub/apps/:name", ctl.hubApp)
	app.Get("/hub/apps/:name/status", ctl.hubAppStatus)
	app.Get("/hub/apps/:name/logs", ctl.hubAppLogs)
	app.Get("/hub/capabilities", ctl.hubCapabilities)
	app.Get("/hub/capabilities/:type", ctl.hubCapability)
	app.Get("/hub/health", ctl.hubHealth)
	return app
}

func TestHubApps_EmptyRegistry(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "empty registry returns empty list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homelab.DefaultRegistry.Replace(nil)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/apps", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Success, r.Status)
		})
	}
}

func TestHubApps_WithApps(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "registry with apps returns app list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := []homelab.App{
				{Name: "app1", Path: "/path/app1"},
				{Name: "app2", Path: "/path/app2"},
			}
			homelab.DefaultRegistry.Replace(apps)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/apps", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Success, r.Status)
		})
	}
}

func TestHubApp_NotFound(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "non-existent app returns error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homelab.DefaultRegistry.Replace(nil)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/apps/nonexistent", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

func TestHubApp_Found(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "existing app returns app data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := []homelab.App{
				{Name: "myapp", Path: "/path/myapp", Status: homelab.AppStatusRunning},
			}
			homelab.DefaultRegistry.Replace(apps)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/apps/myapp", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Success, r.Status)
		})
	}
}

func TestHubAppStatus_NotFound(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "status for non-existent app returns error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homelab.DefaultRegistry.Replace(nil)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/apps/nonexistent/status", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

func TestHubAppLogs_NotFound(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "logs for non-existent app returns error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homelab.DefaultRegistry.Replace(nil)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/apps/nonexistent/logs", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

func TestHubCapabilities_EmptyRegistry(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "empty capability registry returns empty list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/capabilities", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Success, r.Status)
		})
	}
}

func TestHubCapability_NotFound(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "non-existent capability returns error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/capabilities/unknown", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestHubCapability_Found(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "existing capability returns descriptor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := hub.Descriptor{
				Type:        hub.CapBookmark,
				Backend:     "linkding",
				Description: "bookmark service",
			}
			_ = hub.Default.Register(desc)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/capabilities/bookmark", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Success, r.Status)
		})
	}
}

func TestHubCapability_TypeParamCaseSensitive(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "mismatched case returns not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := hub.Descriptor{
				Type:    hub.CapBookmark,
				Backend: "linkding",
			}
			_ = hub.Default.Register(desc)
			app := setupHubTestApp()

			req := httptest.NewRequest(http.MethodGet, "/hub/capabilities/BOOKMARK", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestCheckLifecyclePermission(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		perm      homelab.Permissions
		operation string
		want      bool
	}{
		{
			name:      "status permission returns Status",
			perm:      homelab.Permissions{Status: true},
			operation: "status",
			want:      true,
		},
		{
			name:      "logs permission returns Logs",
			perm:      homelab.Permissions{Logs: true},
			operation: "logs",
			want:      true,
		},
		{
			name:      "start permission returns Start",
			perm:      homelab.Permissions{Start: true},
			operation: "start",
			want:      true,
		},
		{
			name:      "stop permission returns Stop",
			perm:      homelab.Permissions{Stop: false},
			operation: "stop",
			want:      false,
		},
		{
			name:      "restart permission returns Restart",
			perm:      homelab.Permissions{Restart: true},
			operation: "restart",
			want:      true,
		},
		{
			name:      "pull permission returns Pull",
			perm:      homelab.Permissions{Pull: true},
			operation: "pull",
			want:      true,
		},
		{
			name:      "update permission returns Update",
			perm:      homelab.Permissions{Update: true},
			operation: "update",
			want:      true,
		},
		{
			name:      "unknown operation returns false",
			perm:      homelab.Permissions{Status: true},
			operation: "unknown_op",
			want:      false,
		},
		{
			name:      "all permissions false returns false for every operation",
			perm:      homelab.Permissions{},
			operation: "status",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkLifecyclePermission(tt.perm, tt.operation)
			assert.Equal(t, tt.want, got)
		})
	}
}
