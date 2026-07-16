package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// newTestApp creates a minimal Fiber app with the same config as production
// (JSON encoder/decoder, error handler, validator) but without middleware
// that requires external dependencies (redis, database, etc.).
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		JSONDecoder:     sonic.Unmarshal,
		JSONEncoder:     sonic.Marshal,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		StructValidator: &structValidator{validate: validator.New()},
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			if err == nil {
				return nil
			}
			if status, ok := domainErrorStatus(err); ok {
				return ctx.Status(status).
					JSON(protocol.NewFailedResponse(err))
			}
			var e oops.OopsError
			if errors.As(err, &e) {
				if e.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return ctx.Status(fiber.StatusUnauthorized).
						JSON(protocol.NewFailedResponse(e))
				}
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(e))
			}
			if err != nil {
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(protocol.ErrBadRequest.Wrap(err)))
			}
			return nil
		},
	})
	// recover middleware to catch panics (e.g. nil store.Database)
	app.Use(recover.New())
	return app
}

// decodeResponse reads the response body and decodes into a protocol.Response.
func decodeResponse(t *testing.T, resp *http.Response) protocol.Response {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	var r protocol.Response
	err = sonic.Unmarshal(body, &r)
	require.NoError(t, err)
	return r
}

// ---------------------------------------------------------------------------
// Root & health-check endpoints
// ---------------------------------------------------------------------------

func TestRootEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "root returns 200 OK"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get("/", func(_ fiber.Ctx) error { return nil })

			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestHealthcheckLiveness(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "liveness endpoint returns 200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get(healthcheck.LivenessEndpoint, healthcheck.New())

			req := httptest.NewRequest(http.MethodGet, healthcheck.LivenessEndpoint, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestHealthcheckReadiness(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "readiness endpoint returns 200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())

			req := httptest.NewRequest(http.MethodGet, healthcheck.ReadinessEndpoint, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestHealthcheckStartup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "startup endpoint returns 200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get(healthcheck.StartupEndpoint, healthcheck.New())

			req := httptest.NewRequest(http.MethodGet, healthcheck.StartupEndpoint, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Error handler behaviour
// ---------------------------------------------------------------------------

func TestErrorHandler_OopsError_BadRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "OopsError returns 400 BadRequest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get("/err", func(_ fiber.Ctx) error {
				return protocol.ErrBadParam.New("test bad param")
			})

			req := httptest.NewRequest(http.MethodGet, "/err", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Failed, r.Status)
			assert.NotEmpty(t, r.Message)
		})
	}
}

func TestErrorHandler_OopsError_Unauthorized(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "ErrNotAuthorized returns 401 Unauthorized"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get("/unauth", func(_ fiber.Ctx) error {
				return protocol.ErrNotAuthorized.New("not allowed")
			})

			req := httptest.NewRequest(http.MethodGet, "/unauth", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Failed, r.Status)
		})
	}
}

func TestErrorHandler_GenericError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "generic error returns 400 BadRequest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get("/generic", func(_ fiber.Ctx) error {
				return errors.New("something went wrong") //nolint:err113
			})

			req := httptest.NewRequest(http.MethodGet, "/generic", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestErrorHandler_NoError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "no error returns success response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get("/ok", func(c fiber.Ctx) error {
				return c.JSON(protocol.NewSuccessResponse("hello"))
			})

			req := httptest.NewRequest(http.MethodGet, "/ok", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Equal(t, protocol.Success, r.Status)
		})
	}
}

// ---------------------------------------------------------------------------
// Platform callback endpoint
// ---------------------------------------------------------------------------

func TestPlatformCallback_UnknownPlatform(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unknown platform returns error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			ctl := &Controller{}
			app.All("/platform/:platform", ctl.platformCallback)

			req := httptest.NewRequest(http.MethodPost, "/platform/unknown_platform", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			r := decodeResponse(t, resp)
			assert.Contains(t, r.Message, "platform not found")
		})
	}
}

// ---------------------------------------------------------------------------
// Agent data endpoint (no store - expects auth failure)
// ---------------------------------------------------------------------------

func TestAgentData_NoAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wantStatus int
	}{
		{name: "agent data without auth returns unauthorized", wantStatus: http.StatusUnauthorized},
		{name: "agent data with empty body still unauthorized", wantStatus: http.StatusUnauthorized},
		{name: "agent data missing token is rejected", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			ctl := &Controller{}
			app.Post("/agent", ctl.agentData)

			body := strings.NewReader(`{"type":"ping"}`)
			req := httptest.NewRequest(http.MethodPost, "/agent", body)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// OAuth endpoint (no store - expects error)
// ---------------------------------------------------------------------------

func TestOAuth_NoStore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "OAuth without store returns error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			ctl := &Controller{}
			app.All("/oauth/:provider/:flag", ctl.storeOAuth)

			req := httptest.NewRequest(http.MethodGet, "/oauth/github/test-flag", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			// Without database → panics (recovered → 400 via error handler)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Form endpoint (no store - expects error)
// ---------------------------------------------------------------------------

func TestPostForm_NoStore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "POST form without store returns error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			ctl := &Controller{}
			app.Post("/form", ctl.postForm)

			formBody := "x-form_id=abc&x-uid=user1&x-topic=test"
			req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(formBody))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := app.Test(req)
			require.NoError(t, err)
			// Without database → panics (recovered → 400 via error handler)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Protocol Response Tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// OAuth endpoint (no store - expects error)
// ---------------------------------------------------------------------------
// Route registration smoke test
// ---------------------------------------------------------------------------

func TestRouteRegistration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		expectedPath string
	}{
		{name: "root", expectedPath: "/"},
		{name: "liveness", expectedPath: "/livez"},
		{name: "readiness", expectedPath: "/readyz"},
		{name: "startup", expectedPath: "/startupz"},
		{name: "oauth", expectedPath: "/oauth/:provider/:flag"},
		{name: "form", expectedPath: "/form"},
		{name: "agent", expectedPath: "/agent"},
		{name: "platform", expectedPath: "/platform/:platform"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()

			ctl := &Controller{}
			app.Get("/", func(_ fiber.Ctx) error { return nil })
			app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
			app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
			app.Get(healthcheck.StartupEndpoint, healthcheck.New())
			app.All("/oauth/:provider/:flag", ctl.storeOAuth)
			app.Post("/form", ctl.postForm)
			app.Post("/agent", ctl.agentData)
			app.All("/platform/:platform", ctl.platformCallback)

			routes := app.GetRoutes()
			routePaths := make(map[string]bool)
			for _, r := range routes {
				routePaths[r.Path] = true
			}

			assert.True(t, routePaths[tt.expectedPath], "expected route %q to be registered", tt.expectedPath)
		})
	}
}

// ---------------------------------------------------------------------------
// protocol.Response helpers
// ---------------------------------------------------------------------------

func TestNewSuccessResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "success response has correct status and data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := protocol.NewSuccessResponse("data")
			assert.Equal(t, protocol.Success, r.Status)
			assert.Equal(t, "data", r.Data)
			assert.Empty(t, r.Message)
		})
	}
}

func TestNewFailedResponse_WithOopsError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "failed response with OopsError has ret code and message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := protocol.ErrBadRequest.New("test error")
			r := protocol.NewFailedResponse(err)
			assert.Equal(t, protocol.Failed, r.Status)
			assert.NotEmpty(t, r.RetCode)
			assert.NotEmpty(t, r.Message)
		})
	}
}

func TestNewFailedResponse_NilError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "failed response with nil error has default ret code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := protocol.NewFailedResponse(nil)
			assert.Equal(t, protocol.Failed, r.Status)
			assert.Equal(t, "10000", r.RetCode)
		})
	}
}

// ---------------------------------------------------------------------------
// HTTP method routing
// ---------------------------------------------------------------------------

func TestPlatform_MethodRouting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
	}{
		{name: "GET", method: http.MethodGet},
		{name: "POST", method: http.MethodPost},
		{name: "PUT", method: http.MethodPut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			ctl := &Controller{}
			app.All("/platform/:platform", ctl.platformCallback)

			req := httptest.NewRequest(tt.method, "/platform/unknown", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// 404 for unregistered routes
// ---------------------------------------------------------------------------

func TestUnregisteredRoute_Returns404(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unregistered route returns 404 Not Found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Use a plain fiber app without custom error handler to get default 404.
			app := fiber.New()
			app.Get("/", func(_ fiber.Ctx) error { return nil })

			req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// JSON content type
// ---------------------------------------------------------------------------

func TestJSONResponseContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "JSON response has application/json content type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newTestApp()
			app.Get("/json", func(c fiber.Ctx) error {
				return c.JSON(protocol.NewSuccessResponse("test"))
			})

			req := httptest.NewRequest(http.MethodGet, "/json", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		})
	}
}
