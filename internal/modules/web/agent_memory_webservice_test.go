package web

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
)

func TestAgentMemoryFactsAPI(t *testing.T) {
	app, ts := setupTestApp()
	defer func() { handler = moduleHandler{}; config = configType{} }()

	_, err := ts.UpsertAgentMemoryFact(t.Context(), store.AgentMemoryFactUpsert{
		Scope: "my-pipeline", Key: "pref.lang", Value: "en", Pinned: false,
	})
	require.NoError(t, err)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "update existing fact",
			method:     http.MethodPut,
			path:       "/service/web/agent-memory/facts",
			body:       `{"scope":"my-pipeline","key":"pref.lang","value":"zh","pinned":true}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "create missing fact rejected",
			method:     http.MethodPut,
			path:       "/service/web/agent-memory/facts",
			body:       `{"scope":"my-pipeline","key":"new.only","value":"x","pinned":false}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "list facts",
			method:     http.MethodGet,
			path:       "/service/web/agent-memory/facts?scope=my-pipeline",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete fact",
			method:     http.MethodDelete,
			path:       "/service/web/agent-memory/facts?scope=my-pipeline&key=pref.lang",
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader = http.NoBody
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestAgentMemoryFactsFormSaveAndDelete(t *testing.T) {
	app, ts := setupTestApp()
	defer func() { handler = moduleHandler{}; config = configType{} }()

	_, err := ts.UpsertAgentMemoryFact(t.Context(), store.AgentMemoryFactUpsert{
		Scope: "default", Key: "pref.tz", Value: "UTC", Pinned: false,
	})
	require.NoError(t, err)

	tests := []struct {
		name       string
		path       string
		form       url.Values
		wantStatus int
		wantBody   string
	}{
		{
			name: "save updates value and pin",
			path: "/service/web/agent-memory/facts/save",
			form: url.Values{
				"scope":  {"default"},
				"key":    {"pref.tz"},
				"value":  {"Asia/Shanghai"},
				"pinned": {"on"},
			},
			wantStatus: http.StatusOK,
			wantBody:   "Asia/Shanghai",
		},
		{
			name: "delete removes fact",
			path: "/service/web/agent-memory/facts/delete",
			form: url.Values{
				"scope": {"default"},
				"key":   {"pref.tz"},
			},
			wantStatus: http.StatusOK,
			wantBody:   "No facts in scope",
		},
		{
			name: "save missing fact toasts",
			path: "/service/web/agent-memory/facts/save",
			form: url.Values{
				"scope": {"default"},
				"key":   {"missing"},
				"value": {"x"},
			},
			wantStatus: http.StatusNoContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.form.Encode()))
			req.Header.Set("Cookie", "accessToken=test-token")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), tt.wantBody)
			}
		})
	}
}

func TestAgentMemoryFactsUnauthenticated(t *testing.T) {
	app, _ := setupTestApp()
	defer func() { handler = moduleHandler{}; config = configType{} }()

	tests := []struct {
		name string
		path string
	}{
		{name: "list requires auth", path: "/service/web/agent-memory/facts?scope=test"},
		{name: "page requires auth", path: "/service/web/agent-memory"},
		{name: "summaries require auth", path: "/service/web/agent-session-summaries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
		})
	}
}
