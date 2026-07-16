package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestHomelabRegistryPage(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders registry page", wantStatus: http.StatusOK, wantContains: "Homelab Registry"},
		{name: "renders with empty state when no apps", wantStatus: http.StatusOK, wantContains: "No apps discovered"},
		{name: "page has correct title", wantStatus: http.StatusOK, wantContains: "Registry — Flowbot"},
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
			req := httptest.NewRequest(http.MethodGet, "/service/web/homelab", http.NoBody)
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

func TestHomelabRegistryDetailPageNotFound(t *testing.T) {
	tests := []struct {
		name       string
		appName    string
		wantStatus int
	}{
		{name: "non-existent app returns 404", appName: "nonexistent", wantStatus: http.StatusNotFound},
		{name: "unknown app name returns 404", appName: "missing-app", wantStatus: http.StatusNotFound},
		{name: "non-registered app returns 404", appName: "random-name", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/homelab/"+tt.appName, http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestHomelabRegistryUnauthenticated(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "GET /homelab redirects to login", method: http.MethodGet, path: "/service/web/homelab"},
		{name: "GET /homelab/detail redirects to login", method: http.MethodGet, path: "/service/web/homelab/someapp"},
		{name: "authenticated pages render with valid token", method: http.MethodGet, path: "/service/web/homelab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
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

func TestHomelabRegistryRescan(t *testing.T) {
	tests := []struct {
		name            string
		withAuth        bool
		wantStatus      int
		wantHXRedirect  string
		checkHXRedirect bool
	}{
		{
			name:            "rescan returns OK with HX-Redirect",
			withAuth:        true,
			wantStatus:      http.StatusOK,
			wantHXRedirect:  "/service/web/homelab",
			checkHXRedirect: false,
		},
		{
			name:            "rescan unauthenticated redirects",
			withAuth:        false,
			wantStatus:      http.StatusSeeOther,
			wantHXRedirect:  "",
			checkHXRedirect: false,
		},
		{
			name:            "rescan triggers HX-Redirect header",
			withAuth:        true,
			wantStatus:      http.StatusOK,
			wantHXRedirect:  "/service/web/homelab",
			checkHXRedirect: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRunRescan := homelab.LoadRunRescan()
			homelab.SetRunRescan(func() error { return nil })
			defer homelab.SetRunRescan(oldRunRescan)

			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodPost, "/service/web/homelab/rescan", http.NoBody)
			if tt.withAuth {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.checkHXRedirect {
				if got := resp.Header.Get("HX-Redirect"); got != tt.wantHXRedirect {
					t.Errorf("want HX-Redirect %q, got %q", tt.wantHXRedirect, got)
				}
			}
		})
	}
}
