package automate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/module"
)

func resetAutomateModuleState(t *testing.T) {
	t.Helper()
	handler = moduleHandler{}
	config = configType{}
}

func TestModuleRegister(t *testing.T) {
	t.Parallel()
	module.Unregister(Name)
	t.Cleanup(func() { module.Unregister(Name) })
	require.NotPanics(t, func() {
		Register()
	})
	assert.Equal(t, "automate", Name)
}

func TestModuleInit(t *testing.T) {
	tests := []struct {
		name   string
		conf   json.RawMessage
		wantOn bool
		twice  bool
	}{
		{name: "defaults enabled", conf: nil, wantOn: true},
		{name: "can disable", conf: json.RawMessage(`{"enabled":false}`), wantOn: false},
		{name: "e2e is idempotent", conf: nil, wantOn: true, twice: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAutomateModuleState(t)
			t.Cleanup(func() { resetAutomateModuleState(t) })
			require.NoError(t, InitForE2E(tt.conf))
			if tt.twice {
				require.NoError(t, InitForE2E(tt.conf))
			}
			assert.Equal(t, tt.wantOn, handler.IsReady())
		})
	}
}

func TestModuleInitDefaultsEnabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		conf   string
		wantOn bool
	}{
		{name: "empty config enables", conf: `{}`, wantOn: true},
		{name: "explicit enabled true", conf: `{"enabled":true}`, wantOn: true},
		{name: "explicit enabled false", conf: `{"enabled":false}`, wantOn: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := configType{Enabled: true}
			var raw map[string]any
			require.NoError(t, sonic.Unmarshal([]byte(tt.conf), &raw))
			require.NoError(t, sonic.Unmarshal([]byte(tt.conf), &cfg))
			if _, ok := raw["enabled"]; !ok {
				cfg.Enabled = true
			}
			assert.Equal(t, tt.wantOn, cfg.Enabled)
		})
	}
}

func TestWebserviceRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wantMethod string
		path       string
		ruleSet    string
	}{
		{name: "functions apply", ruleSet: "functions", path: "/apply", wantMethod: "POST"},
		{name: "pipeline apply", ruleSet: "pipeline", path: "/apply", wantMethod: "POST"},
		{name: "workflow apply", ruleSet: "workflow", path: "/apply", wantMethod: "POST"},
		{name: "workflow list", ruleSet: "workflow", path: "/list", wantMethod: "GET"},
		{name: "workflow run", ruleSet: "workflow", path: "/run", wantMethod: "POST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rules = functionsRules
			switch tt.ruleSet {
			case "pipeline":
				rules = pipelineRules
			case "workflow":
				rules = workflowRules
			}
			paths := make(map[string]string, len(rules))
			for _, r := range rules {
				paths[r.Path] = r.Method
			}
			require.Equal(t, tt.wantMethod, paths[tt.path])
		})
	}
}

func TestWebserviceRegistersWithoutInit(t *testing.T) {
	t.Parallel()
	h := moduleHandler{}
	require.False(t, h.IsReady())
	app := fiber.New()
	require.NotPanics(t, func() {
		h.Webservice(app)
	})
}

func TestMountForE2E(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "functions apply", method: http.MethodPost, path: "/service/automate/functions/apply"},
		{name: "pipeline list", method: http.MethodGet, path: "/service/automate/pipeline/list"},
		{name: "workflow run", method: http.MethodPost, path: "/service/automate/workflow/run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			require.NotPanics(t, func() {
				MountForE2E(app)
			})
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}
