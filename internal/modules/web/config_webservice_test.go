package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

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
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs", http.NoBody)
			addWebAuth(req)
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
		{name: "renders empty state", storeConfigs: []model.ConfigItem{}, wantStatus: http.StatusOK, wantContains: "No configs yet"},
		{name: "renders multiple rows", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1"), createTestConfig("u2", "t2", "k2")}, wantStatus: http.StatusOK, wantContains: "k2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/list", http.NoBody)
			addWebAuth(req)
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
		configs    []model.ConfigItem
		delErr     error
		wantStatus int
		wantEmpty  bool
		wantBody   string
		wantHX     string
	}{
		{
			name:       "delete with remaining configs returns empty body",
			delErr:     nil,
			configs:    []model.ConfigItem{{ID: 2, UID: "other"}},
			wantStatus: http.StatusOK,
			wantEmpty:  true,
		},
		{
			name:       "delete last config shows empty state row",
			delErr:     nil,
			wantStatus: http.StatusOK,
			wantBody:   `configs-empty`,
		},
		{
			name:       "delete returns toast on store error",
			delErr:     fmt.Errorf("db down"),
			wantStatus: http.StatusNoContent,
			wantHX:     "Failed to delete config",
		},
		{
			name:       "delete missing returns toast",
			delErr:     types.ErrNotFound,
			wantStatus: http.StatusNoContent,
			wantHX:     "Config not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.configs
			if tt.delErr != nil {
				ts.delConfigFn = func(_ types.Uid, _ string, _ string) error { return tt.delErr }
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodDelete, "/service/web/configs/u1/t1/k1", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if tt.wantEmpty && len(body) != 0 {
				t.Errorf("want empty body, got %q", string(body))
			}
			if tt.wantBody != "" && !strings.Contains(string(body), tt.wantBody) {
				t.Errorf("want body containing %q, got %q", tt.wantBody, string(body))
			}
			if tt.wantHX != "" && !strings.Contains(resp.Header.Get("HX-Trigger"), tt.wantHX) {
				t.Errorf("want HX-Trigger containing %q, got %q", tt.wantHX, resp.Header.Get("HX-Trigger"))
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
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/u1/t1/k1", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestNewConfigFormIncludesCleanup(t *testing.T) {
	tests := []struct {
		name          string
		wantStatus    int
		wantContains  string
		wantOOBDelete bool
	}{
		{
			name:          "new config form includes cleanup for existing forms",
			wantStatus:    http.StatusOK,
			wantContains:  `id="config-form-new"`,
			wantOOBDelete: true,
		},
		{
			name:          "new config form returns fragment not full page",
			wantStatus:    http.StatusOK,
			wantContains:  `hx-post="/service/web/configs"`,
			wantOOBDelete: true,
		},
		{
			name:          "new config form is a table row element",
			wantStatus:    http.StatusOK,
			wantContains:  `<tr`,
			wantOOBDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/new", http.NoBody)
			addWebAuth(req)
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			if !strings.Contains(bodyStr, tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}

			if tt.wantOOBDelete {
				hasOOB := strings.Contains(bodyStr, `hx-swap-oob`) && strings.Contains(bodyStr, `"delete"`)
				if !hasOOB {
					t.Errorf("expected OOB delete for config-form-new, got body: %s", bodyStr)
				}
			}
		})
	}
}

func TestCreateConfig(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		setConfigFn      func(uid types.Uid, topic, key string, value types.KV) error
		wantStatus       int
		wantBodyContains string
		wantValue        types.KV
		wantHX           string
	}{
		{
			name:             "valid JSON object creates config successfully",
			body:             "uid=u1&topic=t1&key=k1&value=%7B%22enabled%22%3Atrue%7D",
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{"enabled": true},
		},
		{
			name:             "number value auto-wraps into JSON object successfully",
			body:             "uid=u1&topic=t1&key=k1&value=42",
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{"value": float64(42)},
			wantHX:           "Config saved",
		},
		{
			name:             "invalid JSON value returns 422 with invalid JSON error",
			body:             "uid=u1&topic=t1&key=k1&value=not-json",
			wantStatus:       http.StatusUnprocessableEntity,
			wantBodyContains: "Invalid JSON",
		},
		{
			name:             "empty JSON object value creates config successfully",
			body:             "uid=u1&topic=t1&key=k1&value=%7B%7D",
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{},
			wantHX:           `"type":"success"`,
		},
		{
			name:             "empty value field creates config successfully",
			body:             "uid=u1&topic=t1&key=k1&value=",
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{},
		},
		{
			name:             "missing uid returns 422 with required error",
			body:             "uid=&topic=t1&key=k1&value=%7B%7D",
			wantStatus:       http.StatusUnprocessableEntity,
			wantBodyContains: "UID is required",
		},
		{
			name:             "missing key returns 422 with required error",
			body:             "uid=u1&topic=t1&key=&value=%7B%7D",
			wantStatus:       http.StatusUnprocessableEntity,
			wantBodyContains: "Key is required",
		},
		{
			name:        "store error returns toast",
			body:        "uid=u1&topic=t1&key=k1&value=%7B%7D",
			setConfigFn: func(_ types.Uid, _ string, _ string, _ types.KV) error { return fmt.Errorf("db down") },
			wantStatus:  http.StatusNoContent,
			wantHX:      "Could not save config",
		},
		{
			name:             "JSON string value auto-wraps into JSON object successfully",
			body:             "uid=u1&topic=t1&key=k1&value=%22hello%22",
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{"value": "hello"},
		},
		{
			name:             "JSON array value auto-wraps into JSON object successfully",
			body:             "uid=u1&topic=t1&key=k1&value=%5B1%2C2%2C3%5D",
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{"value": []any{float64(1), float64(2), float64(3)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			var setValue types.KV
			ts.setConfigFn = func(uid types.Uid, topic, key string, value types.KV) error {
				setValue = value
				if tt.setConfigFn != nil {
					return tt.setConfigFn(uid, topic, key, value)
				}
				return nil
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodPost, "/service/web/configs", strings.NewReader(tt.body))
			addWebAuth(req)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBodyContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBodyContains) {
					t.Errorf("want body containing %q, got %s", tt.wantBodyContains, string(body))
				}
			}
			if tt.wantHX != "" && !strings.Contains(resp.Header.Get("HX-Trigger"), tt.wantHX) {
				t.Errorf("want HX-Trigger containing %q, got %q", tt.wantHX, resp.Header.Get("HX-Trigger"))
			}
			if tt.wantValue != nil && !assert.ObjectsAreEqual(tt.wantValue, setValue) {
				t.Errorf("want value %v, got %v", tt.wantValue, setValue)
			}
		})
	}
}

func TestUpdateConfig(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		body             string
		getConfigFn      func(uid types.Uid, topic, key string) (types.KV, error)
		setConfigFn      func(uid types.Uid, topic, key string, value types.KV) error
		wantStatus       int
		wantBodyContains string
		wantValue        types.KV
		wantHX           string
	}{
		{
			name:             "valid JSON object updates config successfully",
			path:             "/service/web/configs/u1/t1/k1",
			body:             "value=%7B%22enabled%22%3Atrue%7D",
			getConfigFn:      func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"old": "value"}, nil },
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{"enabled": true},
			wantHX:           "Config saved",
		},
		{
			name:             "number value auto-wraps into JSON object successfully",
			path:             "/service/web/configs/u1/t1/k1",
			body:             "value=42",
			getConfigFn:      func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"old": "value"}, nil },
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{"value": float64(42)},
			wantHX:           `"type":"success"`,
		},
		{
			name:             "invalid JSON value returns 422 with invalid JSON error",
			path:             "/service/web/configs/u1/t1/k1",
			body:             "value=not-json",
			getConfigFn:      func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"old": "value"}, nil },
			wantStatus:       http.StatusUnprocessableEntity,
			wantBodyContains: "Invalid JSON",
		},
		{
			name:             "empty value field updates config with empty map",
			path:             "/service/web/configs/u1/t1/k1",
			body:             "value=",
			getConfigFn:      func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"old": "value"}, nil },
			wantStatus:       http.StatusOK,
			wantBodyContains: "k1",
			wantValue:        types.KV{},
		},
		{
			name:        "store error returns toast",
			path:        "/service/web/configs/u1/t1/k1",
			body:        "value=%7B%7D",
			getConfigFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"old": "value"}, nil },
			setConfigFn: func(_ types.Uid, _ string, _ string, _ types.KV) error { return fmt.Errorf("db down") },
			wantStatus:  http.StatusNoContent,
			wantHX:      "Could not save config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.getConfigFn = tt.getConfigFn
			var setValue types.KV
			ts.setConfigFn = func(uid types.Uid, topic, key string, value types.KV) error {
				setValue = value
				if tt.setConfigFn != nil {
					return tt.setConfigFn(uid, topic, key, value)
				}
				return nil
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodPut, tt.path, strings.NewReader(tt.body))
			addWebAuth(req)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBodyContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBodyContains) {
					t.Errorf("want body containing %q, got %s", tt.wantBodyContains, string(body))
				}
			}
			if tt.wantHX != "" && !strings.Contains(resp.Header.Get("HX-Trigger"), tt.wantHX) {
				t.Errorf("want HX-Trigger containing %q, got %q", tt.wantHX, resp.Header.Get("HX-Trigger"))
			}
			if tt.wantValue != nil && !assert.ObjectsAreEqual(tt.wantValue, setValue) {
				t.Errorf("want value %v, got %v", tt.wantValue, setValue)
			}
		})
	}
}
