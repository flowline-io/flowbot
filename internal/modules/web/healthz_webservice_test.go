package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/views/partials"
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
		{
			name:         "full page defers capability probes",
			cookie:       "valid-test-token",
			wantStatus:   http.StatusOK,
			wantContains: `hx-get="/service/web/healthz/capabilities"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearHealthzSnapshot()
			app, _ := setupTestApp(t)
			defer func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
				clearHealthzSnapshot()
			}()

			req := httptest.NewRequest(http.MethodGet, "/service/web/healthz", http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
				AttachCSRFForTest(req)
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

func TestHealthzCapabilitiesPartial(t *testing.T) {
	clearHealthzSnapshot()
	app, _ := setupTestApp(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		clearHealthzSnapshot()
	}()

	req := httptest.NewRequest(http.MethodGet, "/service/web/healthz/capabilities", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	AttachCSRFForTest(req)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want status 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Capability Status") {
		t.Errorf("want capability status partial, got %s", body)
	}
	if strings.Contains(string(body), "<!DOCTYPE html>") {
		t.Errorf("capabilities partial must not be a full document")
	}
}

func TestGatherHealthzDataSnapshotCache(t *testing.T) {
	tests := []struct {
		name      string
		seed      partials.HealthzData
		age       time.Duration
		wantReuse bool
	}{
		{
			name:      "returns cached snapshot within TTL",
			seed:      partials.HealthzData{HeapAlloc: 1, PostgresOk: true},
			age:       time.Second,
			wantReuse: true,
		},
		{
			name:      "serves stale snapshot while revalidating",
			seed:      partials.HealthzData{HeapAlloc: 1, PostgresOk: true},
			age:       healthzSnapshotTTL + time.Second,
			wantReuse: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearHealthzSnapshot()
			defer clearHealthzSnapshot()

			healthzSnapshotMu.Lock()
			healthzSnapshot = tt.seed
			healthzSnapshotAt = time.Now().Add(-tt.age)
			healthzSnapshotMu.Unlock()

			got := gatherHealthzData(context.Background())
			if tt.wantReuse {
				if got.HeapAlloc != tt.seed.HeapAlloc {
					t.Errorf("want cached HeapAlloc %d, got %d", tt.seed.HeapAlloc, got.HeapAlloc)
				}
				return
			}
			if got.HeapAlloc == tt.seed.HeapAlloc {
				t.Errorf("want fresh snapshot, still got seeded HeapAlloc %d", got.HeapAlloc)
			}
		})
	}
}

func TestGatherCapabilityHealthSnapshotCache(t *testing.T) {
	clearHealthzSnapshot()
	defer clearHealthzSnapshot()

	seed := []partials.HealthzCap{{Type: "demo", Status: "healthy"}}
	healthzCapMu.Lock()
	healthzCapSnapshot = seed
	healthzCapSnapshotAt = time.Now().Add(-(healthzCapCacheTTL + time.Second))
	healthzCapMu.Unlock()

	got := gatherCapabilityHealth(context.Background())
	if len(got) != 1 || got[0].Type != "demo" || got[0].Status != "healthy" {
		t.Fatalf("want stale capability snapshot, got %+v", got)
	}
}
