package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestChatAgentPermissionsPageUnauthenticated(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
	}{
		{name: "redirects to login without cookie", wantStatus: http.StatusSeeOther},
		{name: "blocks anonymous access", wantStatus: http.StatusSeeOther},
		{name: "requires web authentication", wantStatus: http.StatusSeeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			req := httptest.NewRequest(http.MethodGet, "/service/web/chatagent-permissions", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestChatAgentPermissionsPageAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		contains   string
	}{
		{name: "renders form page", wantStatus: http.StatusOK, contains: "Chat Agent Permissions"},
		{name: "includes general permissions table", wantStatus: http.StatusOK, contains: "General Permissions"},
		{name: "includes advanced json editor", wantStatus: http.StatusOK, contains: "Advanced JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			chatagent.ResetPermissionCacheForTest()
			req := httptest.NewRequest(http.MethodGet, "/service/web/chatagent-permissions", http.NoBody)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.contains)
		})
	}
}

func TestChatAgentPermissionsSaveForm(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantSaved  bool
		wantErr    string
	}{
		{
			name: "form mode saves simple override",
			form: url.Values{
				"submit_mode":     {"form"},
				"perm[websearch]": {"allow"},
				"perm[skill]":     {"inherit"},
			},
			wantStatus: http.StatusSeeOther,
			wantSaved:  true,
		},
		{
			name: "form mode rejects broad bash pattern",
			form: url.Values{
				"submit_mode":                      {"form"},
				"perm[bash][patterns][0][pattern]": {"*"},
				"perm[bash][patterns][0][action]":  {"ask"},
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "too broad",
		},
		{
			name: "form mode inherit all clears overrides",
			form: url.Values{
				"submit_mode":     {"form"},
				"perm[websearch]": {"inherit"},
				"perm[skill]":     {"inherit"},
			},
			wantStatus: http.StatusSeeOther,
			wantSaved:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			chatagent.ResetPermissionCacheForTest()
			var saved permission.Config
			ts.setConfigFn = func(_ types.Uid, topic, key string, value types.KV) error {
				require.Equal(t, chatagent.PermissionTopic, topic)
				require.Equal(t, chatagent.PermissionKey, key)
				data, err := sonic.Marshal(value)
				require.NoError(t, err)
				cfg, err := permission.ParseConfig(data)
				require.NoError(t, err)
				saved = cfg
				return nil
			}

			req := httptest.NewRequest(http.MethodPost, "/service/web/chatagent-permissions", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus == http.StatusBadRequest {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				if tt.wantErr != "" {
					assert.Contains(t, string(body), tt.wantErr)
				}
				return
			}
			if tt.wantSaved {
				_, ok := saved["websearch"]
				assert.True(t, ok)
				return
			}
			assert.Empty(t, saved)
		})
	}
}

func TestChatAgentPermissionsSaveJSON(t *testing.T) {
	tests := []struct {
		name       string
		rules      string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "json mode saves valid user override",
			rules:      `{"websearch":"allow"}`,
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "json mode rejects wildcard key",
			rules:      `{"*":"ask"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "cannot be overridden",
		},
		{
			name:       "json mode rejects invalid json",
			rules:      `{bad`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "Invalid permission JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			chatagent.ResetPermissionCacheForTest()
			var saved bool
			ts.setConfigFn = func(_ types.Uid, topic, key string, _ types.KV) error {
				require.Equal(t, chatagent.PermissionTopic, topic)
				require.Equal(t, chatagent.PermissionKey, key)
				saved = true
				return nil
			}

			form := url.Values{
				"submit_mode": {"json"},
				"rules":       {tt.rules},
			}
			req := httptest.NewRequest(http.MethodPost, "/service/web/chatagent-permissions", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus == http.StatusBadRequest {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				if tt.wantErr != "" {
					assert.Contains(t, string(body), tt.wantErr)
				}
				return
			}
			assert.True(t, saved)
		})
	}
}
