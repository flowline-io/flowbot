package example

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

func TestModuleProperties(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{name: "Name should equal example", test: func(t *testing.T) {
			assert.Equal(t, "example", Name)
		}},
		{name: "Name should not be empty", test: func(t *testing.T) {
			assert.NotEmpty(t, Name)
		}},
		{name: "handler should embed module.Base", test: func(t *testing.T) {
			assert.Implements(t, (*module.Handler)(nil), &handler)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		config  configType
		rawJSON json.RawMessage
		preInit bool
		wantErr bool
		ready   bool
	}{
		{name: "enabled config", config: configType{Enabled: true}, wantErr: false, ready: true},
		{name: "disabled config", config: configType{Enabled: false}, wantErr: false, ready: false},
		{name: "invalid JSON", rawJSON: json.RawMessage(`{invalid`), wantErr: true, ready: false},
		{name: "already initialized", rawJSON: json.RawMessage(`{"enabled":true}`), preInit: true, wantErr: true, ready: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preInit {
				handler = moduleHandler{initialized: true}
			} else {
				handler = moduleHandler{}
			}
			var data json.RawMessage
			if tt.rawJSON != nil {
				data = tt.rawJSON
			} else {
				d, _ := sonic.Marshal(tt.config)
				data = d
			}
			err := handler.Init(data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.ready, handler.IsReady())
			}
		})
	}
}

func TestRules_Validity(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{name: "command rules should contain all expected defines", test: func(t *testing.T) {
			assert.NotEmpty(t, commandRules)
			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}
			assert.Contains(t, defines, "id")
			assert.Contains(t, defines, "form test")
			assert.Contains(t, defines, "event test")
		}},
		{name: "all command rules should have non-nil handlers", test: func(t *testing.T) {
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		}},
		{name: "form rules should define example_form rule", test: func(t *testing.T) {
			assert.NotEmpty(t, formRules)
			found := false
			for _, r := range formRules {
				if r.Id == exampleFormID {
					found = true
					assert.NotEmpty(t, r.Title)
					assert.NotEmpty(t, r.Field)
					assert.NotNil(t, r.Handler)
				}
			}
			assert.True(t, found, "example_form rule should be defined")
		}},
		{name: "webservice rules should not be empty", test: func(t *testing.T) {
			assert.NotEmpty(t, webserviceRules)
		}},
		{name: "webhook rules should not be empty", test: func(t *testing.T) {
			assert.NotEmpty(t, webhookRules)
		}},
		{name: "Rules() should return all 4 rulesets", test: func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 4)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestWebserviceEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET /example without auth returns unauthorized",
			method:     "GET",
			path:       "/service/example/example",
			wantStatus: 401,
		},
		{
			name:       "GET /get without auth returns unauthorized",
			method:     "GET",
			path:       "/service/example/get",
			wantStatus: 401,
		},
		{
			name:       "POST /webhook/example returns 202 without auth",
			method:     "POST",
			path:       "/service/example/webhook/example",
			wantStatus: 202,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: func(ctx fiber.Ctx, err error) error {
					if code, ok := mapDomainErrorStatus(err); ok {
						return ctx.Status(code).SendString(err.Error())
					}
					return ctx.Status(fiber.StatusUnauthorized).SendString(err.Error())
				},
			})
			ruleSets := append(webserviceRules, webhookRules...)
			module.Webservice(app, Name, webservice.Ruleset(ruleSets))

			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			_ = app.Shutdown()
		})
	}
}

func mapDomainErrorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, types.ErrInvalidArgument):
		return fiber.StatusBadRequest, true
	case errors.Is(err, types.ErrUnauthorized):
		return fiber.StatusUnauthorized, true
	case errors.Is(err, types.ErrForbidden):
		return fiber.StatusForbidden, true
	case errors.Is(err, types.ErrNotFound):
		return fiber.StatusNotFound, true
	case errors.Is(err, types.ErrAlreadyExists), errors.Is(err, types.ErrConflict):
		return fiber.StatusConflict, true
	case errors.Is(err, types.ErrRateLimited):
		return fiber.StatusTooManyRequests, true
	case errors.Is(err, types.ErrUnavailable):
		return fiber.StatusServiceUnavailable, true
	case errors.Is(err, types.ErrTimeout):
		return fiber.StatusGatewayTimeout, true
	case errors.Is(err, types.ErrNotImplemented):
		return fiber.StatusNotImplemented, true
	case errors.Is(err, types.ErrProvider):
		return fiber.StatusBadGateway, true
	case errors.Is(err, types.ErrInternal):
		return fiber.StatusInternalServerError, true
	default:
		return fiber.StatusInternalServerError, false
	}
}
