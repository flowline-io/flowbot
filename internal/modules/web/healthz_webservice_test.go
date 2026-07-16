package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
)

func TestHealthzPage(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		hxRequest    string
		wantStatus   int
		wantContains string
		notContains  string
	}{
		{
			name:       "unauthenticated redirects to login",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:         "renders full health dashboard page",
			cookie:       "valid-test-token",
			wantStatus:   http.StatusOK,
			wantContains: "System Health",
		},
		{
			name:         "htmx request returns status partial only",
			cookie:       "valid-test-token",
			hxRequest:    "true",
			wantStatus:   http.StatusOK,
			wantContains: "Database Latency",
			notContains:  "<!DOCTYPE html>",
		},
		{
			name:         "full page includes runtime metrics section",
			cookie:       "valid-test-token",
			wantStatus:   http.StatusOK,
			wantContains: "Goroutines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/healthz", http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
			}
			if tt.hxRequest != "" {
				req.Header.Set("HX-Request", tt.hxRequest)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			if tt.wantContains != "" && !strings.Contains(bodyStr, tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
			if tt.notContains != "" && strings.Contains(bodyStr, tt.notContains) {
				t.Errorf("want body NOT containing %q", tt.notContains)
			}
		})
	}
}
