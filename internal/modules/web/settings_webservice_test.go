package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
)

func TestSettingsPage(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		modules      any
		wantStatus   int
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:       "unauthenticated redirects to login",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "renders settings catalog",
			cookie:     "valid-test-token",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Settings",
				`data-testid="settings-page"`,
				`data-testid="settings-table-filter"`,
				"postgres.dsn",
				pkgconfig.MaskedSecret,
			},
		},
		{
			name:   "redacts nested module passwords",
			cookie: "valid-test-token",
			modules: []any{
				map[string]any{
					"name": "web",
					"auth": map[string]any{
						"username": "admin",
						"password": "flowbot-dev-pass",
					},
				},
			},
			wantStatus: http.StatusOK,
			wantContains: []string{
				"modules[0].auth.password",
				pkgconfig.MaskedSecret,
			},
			wantAbsent: []string{"flowbot-dev-pass"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			prev := pkgconfig.App
			pkgconfig.App = pkgconfig.Type{
				Listen:   ":8080",
				Postgres: pkgconfig.PostgresConfig{DSN: "postgres://user:secret@localhost/db"},
				Modules:  tt.modules,
			}
			defer func() { pkgconfig.App = prev }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/settings", http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
				AttachCSRFForTest(req)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			bodyStr := string(body)
			for _, want := range tt.wantContains {
				if !strings.Contains(bodyStr, want) {
					t.Errorf("want body containing %q", want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(bodyStr, absent) {
					t.Errorf("want body without %q", absent)
				}
			}
		})
	}
}
