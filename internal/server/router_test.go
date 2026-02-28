package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			var e oops.OopsError
			if errors.As(err, &e) {
				if e.Code() == oops.OopsError(protocol.ErrNotAuthorized).Code() {
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
	err = json.Unmarshal(body, &r)
	require.NoError(t, err)
	return r
}

// ---------------------------------------------------------------------------
// Root & health-check endpoints
// ---------------------------------------------------------------------------

func TestRootEndpoint(t *testing.T) {
	app := newTestApp()
	app.Get("/", func(c fiber.Ctx) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealthcheckLiveness(t *testing.T) {
	app := newTestApp()
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())

	req := httptest.NewRequest(http.MethodGet, healthcheck.LivenessEndpoint, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealthcheckReadiness(t *testing.T) {
	app := newTestApp()
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())

	req := httptest.NewRequest(http.MethodGet, healthcheck.ReadinessEndpoint, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealthcheckStartup(t *testing.T) {
	app := newTestApp()
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())

	req := httptest.NewRequest(http.MethodGet, healthcheck.StartupEndpoint, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Error handler behaviour
// ---------------------------------------------------------------------------

func TestErrorHandler_OopsError_BadRequest(t *testing.T) {
	app := newTestApp()
	app.Get("/err", func(_ fiber.Ctx) error {
		return protocol.ErrBadParam.New("test bad param")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Equal(t, protocol.Failed, r.Status)
	assert.NotEmpty(t, r.Message)
}

func TestErrorHandler_OopsError_Unauthorized(t *testing.T) {
	app := newTestApp()
	app.Get("/unauth", func(_ fiber.Ctx) error {
		return protocol.ErrNotAuthorized.New("not allowed")
	})

	req := httptest.NewRequest(http.MethodGet, "/unauth", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Equal(t, protocol.Failed, r.Status)
}

func TestErrorHandler_GenericError(t *testing.T) {
	app := newTestApp()
	app.Get("/generic", func(_ fiber.Ctx) error {
		return errors.New("something went wrong") //nolint:err113
	})

	req := httptest.NewRequest(http.MethodGet, "/generic", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestErrorHandler_NoError(t *testing.T) {
	app := newTestApp()
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.JSON(protocol.NewSuccessResponse("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Equal(t, protocol.Success, r.Status)
}

// ---------------------------------------------------------------------------
// Bearer token auth middleware
// ---------------------------------------------------------------------------

func TestBearerTokenAuth_MissingHeader(t *testing.T) {
	app := newTestApp()
	app.Get("/protected", bearerTokenAuth(func(c fiber.Ctx) error {
		return c.JSON(protocol.NewSuccessResponse("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Equal(t, protocol.Failed, r.Status)
	assert.Contains(t, r.Message, "missing authorization header")
}

func TestBearerTokenAuth_InvalidFormat(t *testing.T) {
	app := newTestApp()
	app.Get("/protected", bearerTokenAuth(func(c fiber.Ctx) error {
		return c.JSON(protocol.NewSuccessResponse("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic abc123")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Contains(t, r.Message, "invalid authorization header format")
}

func TestBearerTokenAuth_EmptyToken(t *testing.T) {
	app := newTestApp()
	app.Get("/protected", bearerTokenAuth(func(c fiber.Ctx) error {
		return c.JSON(protocol.NewSuccessResponse("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestBearerTokenAuth_ValidToken_NoConfiguredCheck(t *testing.T) {
	// When config MCP token is empty, any non-empty bearer token should pass.
	app := newTestApp()
	app.Get("/protected", bearerTokenAuth(func(c fiber.Ctx) error {
		token := c.Locals("mcp_token")
		return c.JSON(protocol.NewSuccessResponse(token))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	// When no configured MCP token, it should pass through
	// (validToken == "" means no check needed)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Webhook endpoint (param routing)
// ---------------------------------------------------------------------------

func TestWebhookRoute_NoBot(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/webhook/:flag", ctl.doWebhook)

	req := httptest.NewRequest(http.MethodGet, "/webhook/nonexistent-flag", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// The handler returns ErrNotFound when bot is not found → error handler returns 400
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Equal(t, protocol.Failed, r.Status)
	assert.Contains(t, r.Message, "not found")
}

func TestWebhookRoute_PostNoBot(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/webhook/:flag", ctl.doWebhook)

	body := strings.NewReader(`{"key":"value"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook/unknown-flag", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Platform callback endpoint
// ---------------------------------------------------------------------------

func TestPlatformCallback_UnknownPlatform(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/chatbot/:platform", ctl.platformCallback)

	req := httptest.NewRequest(http.MethodPost, "/chatbot/unknown_platform", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	r := decodeResponse(t, resp)
	assert.Contains(t, r.Message, "platform not found")
}

// ---------------------------------------------------------------------------
// MCP endpoint (bearer auth + bot routing)
// ---------------------------------------------------------------------------

func TestMCPEndpoint_NoAuth(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/mcp/:bot_name", bearerTokenAuth(ctl.mcpHandler))

	req := httptest.NewRequest(http.MethodPost, "/mcp/some-bot", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMCPEndpoint_InvalidAuth(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/mcp/:bot_name", bearerTokenAuth(ctl.mcpHandler))

	req := httptest.NewRequest(http.MethodPost, "/mcp/some-bot", nil)
	req.Header.Set("Authorization", "Token abc")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Agent data endpoint (no store - expects auth failure)
// ---------------------------------------------------------------------------

func TestAgentData_NoAuth(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.Post("/agent", ctl.agentData)

	body := strings.NewReader(`{"type":"ping"}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Without store.Database initialized, the handler panics (recovered → 400 via error handler)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// getPage endpoint (no store - expects error)
// ---------------------------------------------------------------------------

func TestGetPage_NoStore(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.Get("/p/:id", ctl.getPage)

	req := httptest.NewRequest(http.MethodGet, "/p/test-page-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Without database, store.Database is nil → panics (recovered → 400 via error handler)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// OAuth endpoint (no store - expects error)
// ---------------------------------------------------------------------------

func TestOAuth_NoStore(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/oauth/:provider/:flag", ctl.storeOAuth)

	req := httptest.NewRequest(http.MethodGet, "/oauth/github/test-flag", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Without database → panics (recovered → 400 via error handler)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Form endpoint (no store - expects error)
// ---------------------------------------------------------------------------

func TestPostForm_NoStore(t *testing.T) {
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
}

// ---------------------------------------------------------------------------
// RenderPage endpoint (no store - expects error)
// ---------------------------------------------------------------------------

func TestRenderPage_NoStore(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.Get("/page/:id/:flag", ctl.renderPage)

	req := httptest.NewRequest(http.MethodGet, "/page/rule-id/flag-value", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Without database → panics (recovered → 400 via error handler)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Route registration smoke test
// ---------------------------------------------------------------------------

func TestRouteRegistration(t *testing.T) {
	// Verify that common routes are present after handleRoutes call
	// (we can't call handleRoutes directly because it needs admin controller,
	//  so we verify route patterns manually)
	app := newTestApp()

	// Register routes the same way as in production (minus admin routes)
	ctl := &Controller{}
	app.Get("/", func(c fiber.Ctx) error { return nil })
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())
	app.All("/oauth/:provider/:flag", ctl.storeOAuth)
	app.Get("/p/:id", ctl.getPage)
	app.Post("/form", ctl.postForm)
	app.Get("/page/:id/:flag", ctl.renderPage)
	app.Post("/agent", ctl.agentData)
	app.All("/webhook/:flag", ctl.doWebhook)
	app.All("/chatbot/:platform", ctl.platformCallback)
	app.All("/mcp/:bot_name", bearerTokenAuth(ctl.mcpHandler))

	routes := app.GetRoutes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	expectedPaths := []string{
		"/",
		"/livez",
		"/readyz",
		"/startupz",
		"/oauth/:provider/:flag",
		"/p/:id",
		"/form",
		"/page/:id/:flag",
		"/agent",
		"/webhook/:flag",
		"/chatbot/:platform",
		"/mcp/:bot_name",
	}
	for _, p := range expectedPaths {
		assert.True(t, routePaths[p], "expected route %q to be registered", p)
	}
}

// ---------------------------------------------------------------------------
// protocol.Response helpers
// ---------------------------------------------------------------------------

func TestNewSuccessResponse(t *testing.T) {
	r := protocol.NewSuccessResponse("data")
	assert.Equal(t, protocol.Success, r.Status)
	assert.Equal(t, "data", r.Data)
	assert.Empty(t, r.Message)
}

func TestNewFailedResponse_WithOopsError(t *testing.T) {
	err := protocol.ErrBadRequest.New("test error")
	r := protocol.NewFailedResponse(err)
	assert.Equal(t, protocol.Failed, r.Status)
	assert.NotEmpty(t, r.RetCode)
	assert.NotEmpty(t, r.Message)
}

func TestNewFailedResponse_NilError(t *testing.T) {
	r := protocol.NewFailedResponse(nil)
	assert.Equal(t, protocol.Failed, r.Status)
	assert.Equal(t, "10000", r.RetCode)
}

// ---------------------------------------------------------------------------
// HTTP method routing
// ---------------------------------------------------------------------------

func TestWebhook_MethodRouting(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/webhook/:flag", ctl.doWebhook)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/webhook/test-flag", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			// All methods should reach the handler (which returns bot not found)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestChatbot_MethodRouting(t *testing.T) {
	app := newTestApp()
	ctl := &Controller{}
	app.All("/chatbot/:platform", ctl.platformCallback)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/chatbot/unknown", nil)
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
	// Use a plain fiber app without custom error handler to get default 404.
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// JSON content type
// ---------------------------------------------------------------------------

func TestJSONResponseContentType(t *testing.T) {
	app := newTestApp()
	app.Get("/json", func(c fiber.Ctx) error {
		return c.JSON(protocol.NewSuccessResponse("test"))
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

// ---------------------------------------------------------------------------
// hasToolRules
// ---------------------------------------------------------------------------

func TestHasToolRules_UnknownBot(t *testing.T) {
	result := hasToolRules("nonexistent-bot-xyz")
	assert.False(t, result)
}

// ---------------------------------------------------------------------------
// getBotTools
// ---------------------------------------------------------------------------

func TestGetBotTools_UnknownBot(t *testing.T) {
	_, err := getBotTools("nonexistent-bot-xyz", types.Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found or not ready")
}
