package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestViewPage_Render(t *testing.T) {
	app, _, dbClient := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	pageStore := store.NewPageDataStore(dbClient)
	token := "render-token"
	err := pageStore.CreatePageData(context.Background(), token, "text", "Test Title", types.KV{"content": "Hello World"}, "user", nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid text page",
			token:      token,
			wantStatus: http.StatusOK,
			wantBody:   "Hello World",
		},
		{
			name:       "non-existent token shows expired page",
			token:      "no-such-token",
			wantStatus: http.StatusOK,
			wantBody:   "Page not found or expired",
		},
		{
			name:       "empty token returns method not allowed",
			token:      "",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/service/web/view/" + tt.token
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			}
		})
	}
}

func TestViewPage_Unauthenticated(t *testing.T) {
	app, _, _ := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	req := httptest.NewRequest(http.MethodGet, "/service/web/view/any-token", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	loc := resp.Header.Get("Location")
	assert.Contains(t, loc, "/service/web/login")
}

func TestViewPage_ExpiredPage(t *testing.T) {
	app, _, dbClient := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	pageStore := store.NewPageDataStore(dbClient)
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	token := "expired-render-token"
	err := pageStore.CreatePageData(context.Background(), token, "text", "Expired", types.KV{"content": "stale"}, "user", &oneHourAgo)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/service/web/view/"+token, nil)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Page not found or expired")
}

func TestCreateView(t *testing.T) {
	app, _, dbClient := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "create text page",
			body:       `{"type":"text","title":"Hello","data":{"content":"world"}}`,
			wantStatus: http.StatusCreated,
			wantBody:   `"url":"/service/web/view/`,
		},
		{
			name:       "missing type field",
			body:       `{"title":"NoType","data":{}}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `"error"`,
		},
		{
			name:       "invalid JSON body",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `"error"`,
		},
		{
			name:       "create with expires_at",
			body:       `{"type":"text","title":"Timed","data":{"content":"x"},"expires_at":"2099-01-01T00:00:00Z"}`,
			wantStatus: http.StatusCreated,
			wantBody:   `"url":"/service/web/view/`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/service/web/view", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}

	// Cleanup dbClient after all sub-tests (it's used by TestCreateView only via setupTestAppWithDB).
	_ = dbClient
}

func TestDeleteView(t *testing.T) {
	app, _, dbClient := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	pageStore := store.NewPageDataStore(dbClient)
	token := "del-handler-token"
	err := pageStore.CreatePageData(context.Background(), token, "text", "DeleteMe", types.KV{}, "user", nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "delete existing page",
			token:      token,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete non-existent page",
			token:      "no-such-token",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete with empty token returns method not allowed",
			token:      "",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/service/web/view/" + tt.token
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
