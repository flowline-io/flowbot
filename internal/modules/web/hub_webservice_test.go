package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
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
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
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
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/list", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
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
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/"+tt.appName, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
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
			req := httptest.NewRequest(http.MethodPost, "/service/web/hub/test-app/"+tt.action, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
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
			req := httptest.NewRequest(http.MethodGet, "/service/web/hub/"+tt.appName+"/logs/stream", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
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
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.name == "authenticated pages render with valid token" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
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
