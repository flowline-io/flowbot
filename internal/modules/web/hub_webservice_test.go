package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestHubAppsPage(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders apps page", wantStatus: http.StatusOK, wantContains: "Apps"},
		{name: "renders with empty table when no apps", wantStatus: http.StatusOK, wantContains: "No apps discovered"},
		{name: "page has correct title", wantStatus: http.StatusOK, wantContains: "Apps — Flowbot"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			oldApps := homelab.DefaultRegistry.List()
			homelab.DefaultRegistry.Replace(nil)
			defer func() {
				homelab.DefaultRegistry.Replace(oldApps)
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
			}()
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
		})
	}
}

func TestHubAppsList(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders table partial", wantStatus: http.StatusOK, wantContains: "hub-apps-table"},
		{name: "includes htmx trigger", wantStatus: http.StatusOK, wantContains: "hx-trigger=\"every 10s\""},
		{name: "empty state shown when no apps", wantStatus: http.StatusOK, wantContains: "No apps discovered"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			oldApps := homelab.DefaultRegistry.List()
			homelab.DefaultRegistry.Replace(nil)
			defer func() {
				homelab.DefaultRegistry.Replace(oldApps)
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
			}()
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/list", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
		})
	}
}

func TestHubAppDetailPageNotFound(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		wantStatus  int
		wantContent string
	}{
		{name: "non-existent app returns 404", appName: "nonexistent", wantStatus: http.StatusNotFound, wantContent: "app not found"},
		{name: "unknown app name returns 404", appName: "missing-app", wantStatus: http.StatusNotFound, wantContent: "app not found"},
		{name: "non-registered app returns 404", appName: "random-name", wantStatus: http.StatusNotFound, wantContent: "app not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/"+tt.appName, http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContent) {
				t.Errorf("want body containing %q", tt.wantContent)
			}
		})
	}
}

func TestHubAppActionNotFound(t *testing.T) {
	tests := []struct {
		name   string
		action string
	}{
		{name: "start returns 404 for unknown app", action: "start"},
		{name: "stop returns 404 for unknown app", action: "stop"},
		{name: "restart returns 404 for unknown app", action: "restart"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodPost, "/service/web/hub/test-app/"+tt.action, http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("want status %d, got %d", http.StatusNotFound, resp.StatusCode)
			}
		})
	}
}

func TestHubAppLogsSSENotFound(t *testing.T) {
	tests := []struct {
		name       string
		appName    string
		wantStatus int
	}{
		{name: "not found returns 404", appName: "noapp", wantStatus: http.StatusNotFound},
		{name: "empty name returns 404", appName: "", wantStatus: http.StatusNotFound},
		{name: "valid app name but not registered", appName: "testapp", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/"+tt.appName+"/logs/stream", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestHubAppsUnauthenticated(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "GET /hub redirects to login", method: http.MethodGet, path: "/service/web/hub"},
		{name: "GET /hub/list redirects to login", method: http.MethodGet, path: "/service/web/hub/list"},
		{name: "authenticated pages render with valid token", method: http.MethodGet, path: "/service/web/hub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			if tt.name == "authenticated pages render with valid token" {
				addWebAuth(req)
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.name == "authenticated pages render with valid token" {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("want status 200 with token, got %d", resp.StatusCode)
				}
			} else if resp.StatusCode != http.StatusSeeOther {
				t.Errorf("want status %d (redirect), got %d", http.StatusSeeOther, resp.StatusCode)
			}
		})
	}
}

func TestHubCapabilitiesPage(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders capabilities page", wantStatus: http.StatusOK, wantContains: "Capabilities — Flowbot"},
		{name: "includes filter dropdown for type", wantStatus: http.StatusOK, wantContains: "capability-type-filter"},
		{name: "shows empty state when no capabilities", wantStatus: http.StatusOK, wantContains: "No capabilities registered"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/capabilities", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
		})
	}
}

func TestHubCapabilitiesGrid(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders grid partial", wantStatus: http.StatusOK, wantContains: "capability-grid"},
		{name: "empty state shown when no capabilities", wantStatus: http.StatusOK, wantContains: "No capabilities registered"},
		{name: "accepts type query param", wantStatus: http.StatusOK, wantContains: "capability-grid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			url := "/service/web/capabilities/grid"
			if tt.name == "accepts type query param" {
				url = "/service/web/capabilities/grid?type=bookmark"
			}
			req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
		})
	}
}

func TestHubCapabilitiesGridFiltered(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "filter by type shows only matching card",
			wantStatus:   http.StatusOK,
			wantContains: "capability-card-karakeep",
		},
		{
			name:         "filter with match shows grid not empty state",
			wantStatus:   http.StatusOK,
			wantContains: "capability-grid",
		},
		{
			name:         "no match filter shows empty filter message",
			wantStatus:   http.StatusOK,
			wantContains: "No capabilities match these filters",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDefault := hub.Default
			hub.Default = hub.NewRegistry()
			defer func() { hub.Default = oldDefault }()

			hub.Default.Register(hub.Descriptor{
				Type:        hub.CapKarakeep,
				App:         "test-app",
				Description: "Test capability",
				Healthy:     true,
			})

			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			var url string
			switch tt.name {
			case "filter by type shows only matching card":
				url = "/service/web/capabilities/grid?type=karakeep"
			case "filter with match shows grid not empty state":
				url = "/service/web/capabilities/grid?type=karakeep&provider=karakeep"
			case "no match filter shows empty filter message":
				url = "/service/web/capabilities/grid?type=unknown"
			}

			req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q, got: %s", tt.wantContains, string(body))
			}
		})
	}
}

func TestHubCapabilitiesUnauthenticated(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "GET /capabilities redirects to login", method: http.MethodGet, path: "/service/web/capabilities"},
		{name: "GET /capabilities/grid redirects to login", method: http.MethodGet, path: "/service/web/capabilities/grid"},
		{name: "authenticated capabilities page renders OK", method: http.MethodGet, path: "/service/web/capabilities"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			if tt.name == "authenticated capabilities page renders OK" {
				addWebAuth(req)
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.name == "authenticated capabilities page renders OK" {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("want status 200 with token, got %d", resp.StatusCode)
				}
			} else if resp.StatusCode != http.StatusSeeOther {
				t.Errorf("want status %d (redirect), got %d", http.StatusSeeOther, resp.StatusCode)
			}
		})
	}
}

func TestHubLifecycleAction_PermissionDenied(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		perms      homelab.Permissions
		wantStatus int
	}{
		{
			name:       "start denied when Start false",
			action:     "start",
			perms:      homelab.Permissions{},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "stop denied when Stop false",
			action:     "stop",
			perms:      homelab.Permissions{Start: true},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "restart denied when Restart false",
			action:     "restart",
			perms:      homelab.Permissions{Start: true, Stop: true},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			oldApps := homelab.DefaultRegistry.List()
			oldPerms := homelab.DefaultRegistry.Permissions()
			defer func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
				homelab.DefaultRegistry.Replace(oldApps)
				homelab.DefaultRegistry.SetPermissions(oldPerms)
			}()
			homelab.DefaultRegistry.SetPermissions(tt.perms)
			homelab.DefaultRegistry.Replace([]homelab.App{{Name: "perm-app"}})

			req := httptest.NewRequest(http.MethodPost, "/service/web/hub/perm-app/"+tt.action, http.NoBody)
			addWebAuth(req)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("want status %d, got %d body=%s", tt.wantStatus, resp.StatusCode, body)
			}
		})
	}
}

type stubHubRuntime struct {
	statusByName map[string]homelab.AppStatus
	errByName    map[string]error
}

func (s stubHubRuntime) Status(_ context.Context, app homelab.App) (homelab.AppStatus, error) {
	if err, ok := s.errByName[app.Name]; ok {
		return homelab.AppStatusUnknown, err
	}
	if status, ok := s.statusByName[app.Name]; ok {
		return status, nil
	}
	return homelab.AppStatusUnknown, nil
}

func (stubHubRuntime) Logs(context.Context, homelab.App, int) ([]string, error) {
	return nil, nil
}
func (stubHubRuntime) Start(context.Context, homelab.App) error   { return nil }
func (stubHubRuntime) Stop(context.Context, homelab.App) error    { return nil }
func (stubHubRuntime) Restart(context.Context, homelab.App) error { return nil }
func (stubHubRuntime) Pull(context.Context, homelab.App) error    { return nil }
func (stubHubRuntime) Update(context.Context, homelab.App) error  { return nil }

func TestEnrichAppStatuses(t *testing.T) {
	tests := []struct {
		name         string
		apps         []homelab.App
		runtime      stubHubRuntime
		wantStatuses []homelab.AppStatus
	}{
		{
			name: "fills live runtime statuses",
			apps: []homelab.App{
				{Name: "atuin", Status: homelab.AppStatusUnknown},
				{Name: "caddy", Status: homelab.AppStatusUnknown},
			},
			runtime: stubHubRuntime{statusByName: map[string]homelab.AppStatus{
				"atuin": homelab.AppStatusRunning,
				"caddy": homelab.AppStatusStopped,
			}},
			wantStatuses: []homelab.AppStatus{homelab.AppStatusRunning, homelab.AppStatusStopped},
		},
		{
			name: "keeps previous status when runtime errors",
			apps: []homelab.App{
				{Name: "broken", Status: homelab.AppStatusUnknown},
			},
			runtime: stubHubRuntime{errByName: map[string]error{
				"broken": errors.New("docker unavailable"),
			}},
			wantStatuses: []homelab.AppStatus{homelab.AppStatusUnknown},
		},
		{
			name:         "empty input returns empty",
			apps:         nil,
			runtime:      stubHubRuntime{},
			wantStatuses: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := homelab.DefaultRuntime
			homelab.DefaultRuntime = tt.runtime
			defer func() { homelab.DefaultRuntime = prev }()

			got := enrichAppStatuses(context.Background(), tt.apps)
			if len(got) != len(tt.wantStatuses) {
				t.Fatalf("want %d apps, got %d", len(tt.wantStatuses), len(got))
			}
			for i, want := range tt.wantStatuses {
				if got[i].Status != want {
					t.Errorf("app %d status: want %q, got %q", i, want, got[i].Status)
				}
			}
		})
	}
}

func TestHubAppsListShowsRuntimeStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       homelab.AppStatus
		wantContains string
	}{
		{name: "running shows online", status: homelab.AppStatusRunning, wantContains: "online"},
		{name: "stopped shows offline", status: homelab.AppStatusStopped, wantContains: "offline"},
		{name: "unknown shows unknown not error", status: homelab.AppStatusUnknown, wantContains: ">unknown<"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			oldApps := homelab.DefaultRegistry.List()
			prevRuntime := homelab.DefaultRuntime
			homelab.DefaultRegistry.Replace([]homelab.App{{Name: "status-app", Status: homelab.AppStatusUnknown}})
			homelab.DefaultRuntime = stubHubRuntime{statusByName: map[string]homelab.AppStatus{
				"status-app": tt.status,
			}}
			defer func() {
				homelab.DefaultRegistry.Replace(oldApps)
				homelab.DefaultRuntime = prevRuntime
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
			}()

			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/list", http.NoBody)
			addWebAuth(req)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("want status 200, got %d", resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q, got %s", tt.wantContains, body)
			}
			if tt.status == homelab.AppStatusUnknown && strings.Contains(string(body), ">error<") {
				t.Errorf("unknown status must not render as error badge")
			}
		})
	}
}
