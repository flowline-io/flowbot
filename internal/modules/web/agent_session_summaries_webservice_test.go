package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

func TestAgentSessionSummariesListAuthenticated(t *testing.T) {
	app, ts := setupTestApp()
	defer func() { handler = moduleHandler{}; config = configType{} }()

	_, err := ts.UpsertAgentSessionSummaryPending(t.Context(), "sess-sum-1", "default", "Archive topic")
	require.NoError(t, err)
	_, err = ts.ClaimAgentSessionSummaryPending(t.Context(), "web-tok")
	require.NoError(t, err)
	require.NoError(t, ts.MarkAgentSessionSummaryReady(t.Context(), "sess-sum-1", "web-tok", "Archive topic", "Discussed deploy rollback"))

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
		wantAbsent string
	}{
		{
			name:       "page lists ready summary",
			path:       "/service/web/agent-session-summaries",
			wantStatus: http.StatusOK,
			wantBody:   "Archive topic",
		},
		{
			name:       "table search hits summary text",
			path:       "/service/web/agent-session-summaries/list?q=deploy",
			wantStatus: http.StatusOK,
			wantBody:   "deploy rollback",
		},
		{
			name:       "table search miss shows empty state",
			path:       "/service/web/agent-session-summaries/list?q=zzzz-no-hit",
			wantStatus: http.StatusOK,
			wantAbsent: "Archive topic",
			wantBody:   "No session summaries",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			body := string(bodyBytes)
			if tt.wantBody != "" {
				assert.Contains(t, body, tt.wantBody)
			}
			if tt.wantAbsent != "" {
				assert.NotContains(t, body, tt.wantAbsent)
			}
		})
	}
}

func TestAgentSessionSummaryRetry(t *testing.T) {
	restoreLLM := chatagent.DisableSessionSummaryLLMForTest()
	t.Cleanup(func() {
		chatagent.WaitForSessionSummaryGenerationForTest()
		restoreLLM()
	})
	app, _ := setupTestApp()
	defer func() { handler = moduleHandler{}; config = configType{} }()

	tests := []struct {
		name       string
		path       string
		auth       bool
		wantStatus int
	}{
		{
			name:       "retry accepted for session",
			path:       "/service/web/agent-session-summaries/sess-retry-1/retry",
			auth:       true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "retry accepted for another session",
			path:       "/service/web/agent-session-summaries/sess-missing/retry",
			auth:       true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "retry without auth redirects",
			path:       "/service/web/agent-session-summaries/sess-retry-1/retry",
			auth:       false,
			wantStatus: http.StatusSeeOther,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, http.NoBody)
			if tt.auth {
				req.Header.Set("Cookie", "accessToken=test-token")
				AttachCSRFForTest(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
