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

func TestHomePageAuth(t *testing.T) {
	tests := []struct {
		name       string
		cookie     string
		wantStatus int
	}{
		{name: "unauthenticated redirects to login", wantStatus: http.StatusSeeOther},
		{name: "authenticated renders home", cookie: "valid-test-token", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/home", http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestHomeTokenUsage(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		cookie       string
		wantStatus   int
		wantContains string
	}{
		{
			name:       "unauthenticated redirects",
			path:       "/service/web/home/token-usage?range=7d",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:         "authenticated empty usage partial",
			path:         "/service/web/home/token-usage?range=7d&groupBy=model",
			cookie:       "valid-test-token",
			wantStatus:   http.StatusOK,
			wantContains: "token-usage-container",
		},
		{
			name:       "invalid groupBy returns bad request",
			path:       "/service/web/home/token-usage?groupBy=bad",
			cookie:     "valid-test-token",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "since after until returns bad request",
			path:       "/service/web/home/token-usage?since=2026-07-10&until=2026-07-01",
			cookie:     "valid-test-token",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "partial since returns bad request",
			path:       "/service/web/home/token-usage?since=2026-07-01",
			cookie:     "valid-test-token",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, _ := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookie})
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

func TestHomeTokenUsageJSON(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	usageStore := store.NewLLMUsageStore(client)
	require.NoError(t, usageStore.RecordLLMUsage(t.Context(), &types.LLMUsageRecordInput{
		UID: "testuser", Model: "gpt-4o",
		PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
	}))

	req := httptest.NewRequest(http.MethodGet, "/service/web/home/token-usage?range=7d", http.NoBody)
	req.Header.Set("Accept", "application/json")
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats types.TokenUsageStats
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats))
	assert.Equal(t, int64(150), stats.Summary.TotalTokens)
}
