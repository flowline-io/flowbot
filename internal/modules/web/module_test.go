package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestRegister(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "register should not panic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				Register()
			})
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		jsonCfg string
		wantErr bool
	}{
		{
			name:    "enabled true succeeds",
			jsonCfg: `{"enabled": true}`,
			wantErr: false,
		},
		{
			name:    "disabled skips initialization",
			jsonCfg: `{"enabled": false}`,
			wantErr: false,
		},
		{
			name:    "invalid json returns error",
			jsonCfg: `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &moduleHandler{}
			err := h.Init(json.RawMessage(tt.jsonCfg))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Reset handler state for subsequent tests
			handler = moduleHandler{}
			config = configType{}
		})
	}
}

func TestIsReady(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		want        bool
	}{
		{
			name:        "ready after init",
			initialized: true,
			want:        true,
		},
		{
			name:        "not ready before init",
			initialized: false,
			want:        false,
		},
		{
			name:        "not ready when disabled",
			initialized: false,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: tt.initialized}
			assert.Equal(t, tt.want, handler.IsReady())
			handler = moduleHandler{}
		})
	}
}

func TestConfigsPage(t *testing.T) {
	tests := []struct {
		name, wantContains string
		storeConfigs       []model.ConfigItem
		storeErr           error
		wantStatus         int
	}{
		{name: "renders page with configs", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1")}, wantStatus: http.StatusOK, wantContains: "k1"},
		{name: "renders page with empty list", storeConfigs: []model.ConfigItem{}, wantStatus: http.StatusOK, wantContains: "Configs"},
		{name: "store error returns 500", storeErr: fmt.Errorf("db down"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			if tt.storeErr != nil {
				ts.configErr = tt.storeErr
			}
			defer func() { store.Database = nil }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs?accessToken=test", nil)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}

func TestListConfigs(t *testing.T) {
	tests := []struct {
		name, wantContains string
		storeConfigs       []model.ConfigItem
		wantStatus         int
	}{
		{name: "renders config table", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1")}, wantStatus: http.StatusOK, wantContains: "k1"},
		{name: "renders empty state", storeConfigs: []model.ConfigItem{}, wantStatus: http.StatusOK, wantContains: "No configs"},
		{name: "renders multiple rows", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1"), createTestConfig("u2", "t2", "k2")}, wantStatus: http.StatusOK, wantContains: "k2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			defer func() { store.Database = nil }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/list?accessToken=test", nil)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}

func TestDeleteConfig(t *testing.T) {
	tests := []struct {
		name       string
		delErr     error
		wantStatus int
	}{
		{name: "delete returns 200 on success", wantStatus: http.StatusOK},
		{name: "delete returns 500 on store error", delErr: fmt.Errorf("db down"), wantStatus: http.StatusInternalServerError},
		{name: "delete non-existent returns 404", delErr: types.ErrNotFound, wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.delErr != nil {
				ts.delConfigFn = func(_ types.Uid, _ string, _ string) error { return tt.delErr }
			}
			defer func() { store.Database = nil }()
			req := httptest.NewRequest(http.MethodDelete, "/service/web/configs/u1/t1/k1?accessToken=test", nil)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name       string
		getFn      func(uid types.Uid, topic, key string) (types.KV, error)
		wantStatus int
	}{
		{name: "existing config returns row", getFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"v": "foo"}, nil }, wantStatus: http.StatusOK},
		{name: "not found returns 404", getFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return nil, types.ErrNotFound }, wantStatus: http.StatusNotFound},
		{name: "store error returns 500", getFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return nil, fmt.Errorf("db down") }, wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.getConfigFn = tt.getFn
			defer func() { store.Database = nil }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/u1/t1/k1?accessToken=test", nil)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}
