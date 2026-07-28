package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/version"
)

func TestAboutPage(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		wantStatus   int
		wantContains []string
	}{
		{
			name:       "unauthenticated redirects to login",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "renders about page with version and build info",
			cookie:     "valid-test-token",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"About",
				"data-testid=\"about-info\"",
				"data-testid=\"about-version\"",
				version.Buildtags,
				"data-testid=\"about-buildstamp\"",
				version.Buildstamp,
				"data-testid=\"about-go-version\"",
				"data-testid=\"about-platform\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/about", http.NoBody)
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

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			for _, want := range tt.wantContains {
				if !strings.Contains(bodyStr, want) {
					t.Errorf("want body containing %q", want)
				}
			}
		})
	}
}
