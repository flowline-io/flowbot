package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestAnalyticsTokenUsage(t *testing.T) {
	app, _, _ := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	tests := []struct {
		name         string
		path         string
		cookie       string
		hx           bool
		wantStatus   int
		wantContains string
	}{
		{
			name:       "unauthenticated redirects",
			path:       "/service/web/analytics/token-usage?range=7d",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:         "authenticated page",
			path:         "/service/web/analytics/token-usage?range=7d&groupBy=model",
			cookie:       "valid-test-token",
			wantStatus:   http.StatusOK,
			wantContains: "token-usage-container",
		},
		{
			name:         "htmx partial",
			path:         "/service/web/analytics/token-usage?range=7d&groupBy=model",
			cookie:       "valid-test-token",
			hx:           true,
			wantStatus:   http.StatusOK,
			wantContains: "token-usage-container",
		},
		{
			name:       "invalid groupBy returns bad request",
			path:       "/service/web/analytics/token-usage?groupBy=bad",
			cookie:     "valid-test-token",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
				AttachCSRFForTest(req)
			}
			if tt.hx {
				req.Header.Set("HX-Request", "true")
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

func TestAnalyticsTokenUsageJSON(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	usageStore := store.NewLLMUsageStore(client)
	require.NoError(t, usageStore.RecordLLMUsage(t.Context(), &types.LLMUsageRecordInput{
		UID: "testuser", Model: "gpt-4o",
		PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
	}))

	req := httptest.NewRequest(http.MethodGet, "/service/web/analytics/token-usage?range=7d", http.NoBody)
	req.Header.Set("Accept", "application/json")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats types.TokenUsageStats
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats))
	assert.Equal(t, int64(150), stats.Summary.TotalTokens)
}

func TestAnalyticsOverviewPage(t *testing.T) {
	app, _, _ := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	req := httptest.NewRequest(http.MethodGet, "/service/web/analytics?range=7d", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `data-testid="analytics-module"`)
}
