package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
)

func TestHomePageAuth(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		wantStatus   int
		wantContains string
	}{
		{name: "unauthenticated redirects to login", wantStatus: http.StatusSeeOther},
		{name: "authenticated renders home shell", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "home-dashboard-loader"},
		{name: "authenticated SSR dashboard summary", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "home-dashboard-summary"},
		{name: "authenticated shows quick links", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "home-quick-links"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/home", http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
				AttachCSRFForTest(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantContains)
			}
		})
	}
}

func TestHomeDashboardPartial(t *testing.T) {
	tests := []struct {
		name            string
		cookie          string
		wantStatus      int
		wantContains    string
		wantNotContains string
	}{
		{name: "unauthenticated redirects", wantStatus: http.StatusSeeOther},
		{name: "authenticated returns summary", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "home-dashboard-summary"},
		{name: "authenticated returns checklist when empty", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "home-setup-checklist"},
		{name: "authenticated omits quick links from dashboard partial", cookie: "valid-test-token", wantStatus: http.StatusOK, wantNotContains: "home-quick-links"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/home/dashboard", http.NoBody)
			if tt.cookie != "" {
				addWebAuth(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantContains != "" || tt.wantNotContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if tt.wantContains != "" {
					assert.Contains(t, string(body), tt.wantContains)
				}
				if tt.wantNotContains != "" {
					assert.NotContains(t, string(body), tt.wantNotContains)
				}
			}
		})
	}
}

func TestSessionBadge(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		wantStatus   int
		wantContains string
	}{
		{name: "unauthenticated redirects", wantStatus: http.StatusSeeOther},
		{name: "authenticated renders username", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "session-badge"},
		{name: "authenticated shows testuser", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "testuser"},
		{name: "authenticated shows 24h session label", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "24h session"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/session-badge", http.NoBody)
			if tt.cookie != "" {
				addWebAuth(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantContains)
			}
		})
	}
}

func TestApprovalBadge(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		wantStatus   int
		wantContains string
	}{
		{name: "unauthenticated redirects", wantStatus: http.StatusSeeOther},
		{name: "authenticated renders empty badge slot", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "approval-count-badge-empty"},
		{name: "authenticated omits warning chip when idle", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: `data-testid="approval-count-badge-empty"`},
		{name: "authenticated copies empty badge to replica slots", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: `hx-swap-oob="innerHTML:#nav-mobile-agents-approval-badge"`},
		{name: "authenticated copies empty badge to agents menu slot", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: `hx-swap-oob="innerHTML:#nav-agents-approval-badge"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/approval-badge", http.NoBody)
			if tt.cookie != "" {
				addWebAuth(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantContains)
			}
		})
	}
}

func TestInboxBadge(t *testing.T) {
	tests := []struct {
		name         string
		cookie       string
		wantStatus   int
		wantContains string
	}{
		{name: "unauthenticated redirects", wantStatus: http.StatusSeeOther},
		{name: "authenticated renders empty badge slot", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: "inbox-count-badge-empty"},
		{name: "authenticated copies empty badge to mobile slot", cookie: "valid-test-token", wantStatus: http.StatusOK, wantContains: `hx-swap-oob="innerHTML:#nav-mobile-inbox-badge"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/inbox-badge", http.NoBody)
			if tt.cookie != "" {
				addWebAuth(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantContains)
			}
		})
	}
}

func TestHomeTokenUsageRedirectsToAnalytics(t *testing.T) {
	app, _, _ := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	tests := []struct {
		name       string
		path       string
		cookie     string
		wantStatus int
		wantLoc    string
	}{
		{
			name:       "unauthenticated redirects to login",
			path:       "/service/web/home/token-usage?range=7d",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "authenticated redirects to analytics",
			path:       "/service/web/home/token-usage?range=7d&groupBy=model",
			cookie:     "valid-test-token",
			wantStatus: http.StatusFound,
			wantLoc:    "/service/web/analytics/token-usage?range=7d&groupBy=model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
				AttachCSRFForTest(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantLoc != "" {
				assert.Equal(t, tt.wantLoc, resp.Header.Get("Location"))
			}
		})
	}
}
