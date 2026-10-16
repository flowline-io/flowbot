package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

type memoryTokenStore struct {
	mu   sync.Mutex
	rows map[string]route.AccessToken
}

func (m *memoryTokenStore) Get(_ context.Context, flag string) (route.AccessToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.rows[flag]
	if !ok {
		return route.AccessToken{}, types.ErrNotFound
	}
	return p, nil
}

func (m *memoryTokenStore) Set(_ context.Context, flag string, params types.KV, expiredAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rows == nil {
		m.rows = map[string]route.AccessToken{}
	}
	cp := map[string]any{}
	maps.Copy(cp, params)
	m.rows[flag] = route.AccessToken{ID: 1, Flag: flag, Params: cp, ExpiredAt: expiredAt}
	return nil
}

func (m *memoryTokenStore) SetParams(_ context.Context, flag string, params types.KV) error {
	return m.Set(context.Background(), flag, params, time.Now().Add(time.Hour))
}

func (m *memoryTokenStore) Delete(_ context.Context, flag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, flag)
	return nil
}

func TestRegisterMethodAndAuth(t *testing.T) {
	prevEnabled := config.App.MCP.Enabled
	config.App.MCP.Enabled = true
	t.Cleanup(func() { config.App.MCP.Enabled = prevEnabled })

	store := &memoryTokenStore{rows: map[string]route.AccessToken{}}
	route.SetAccessTokenStore(store)
	t.Cleanup(func() { route.SetAccessTokenStore(nil) })

	fullTok := "mcp-full-token"
	require.NoError(t, store.Set(context.Background(), auth.HashToken(fullTok), types.KV{
		"uid": "user-1", "kind": webauth.KindFull, "scopes": []string{auth.ScopeAdmin},
	}, time.Now().Add(time.Hour)))

	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			if err == nil {
				return nil
			}
			if status, ok := domainStatus(err); ok {
				return ctx.Status(status).JSON(protocol.NewFailedResponse(err))
			}
			if fiberErr, ok := errors.AsType[*fiber.Error](err); ok {
				return ctx.Status(fiberErr.Code).JSON(protocol.NewFailedResponse(err))
			}
			if e, ok := errors.AsType[oops.OopsError](err); ok {
				if e.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return ctx.Status(fiber.StatusUnauthorized).JSON(protocol.NewFailedResponse(e))
				}
				return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(e))
			}
			return ctx.Status(fiber.StatusInternalServerError).JSON(protocol.NewFailedResponse(err))
		},
	})
	Register(app, nil)

	t.Run("GET returns 405 without token", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/mcp", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("DELETE returns 405 without token", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodDelete, "/mcp", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("POST without token returns 401", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("POST kind=full returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+fullTok)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func domainStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, types.ErrForbidden):
		return fiber.StatusForbidden, true
	case errors.Is(err, types.ErrUnauthorized):
		return fiber.StatusUnauthorized, true
	default:
		return 0, false
	}
}

func TestStreamableToolCallRequiresProtocolHeaders(t *testing.T) {
	prevEnabled := config.App.MCP.Enabled
	config.App.MCP.Enabled = true
	t.Cleanup(func() { config.App.MCP.Enabled = prevEnabled })

	store := &memoryTokenStore{rows: map[string]route.AccessToken{}}
	route.SetAccessTokenStore(store)
	t.Cleanup(func() { route.SetAccessTokenStore(nil) })

	token := "mcp-protocol-token"
	require.NoError(t, store.Set(context.Background(), auth.HashToken(token), types.KV{
		"uid": "user-1", "kind": "api",
		"scopes": []string{auth.ScopeHubAppsRead, auth.ScopeHubHealthRead},
	}, time.Now().Add(time.Hour)))

	app := fiber.New()
	app.Use(requestid.New())
	Register(app, nil)

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hub.health","arguments":{},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`)

	t.Run("missing protocol headers is 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("matching headers return hub.health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
		req.Header.Set("Mcp-Method", "tools/call")
		req.Header.Set("Mcp-Name", "hub.health")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(raw))
		require.NotEmpty(t, resp.Header.Get("X-Request-ID"))
		assert.Contains(t, string(raw), `"status"`)
		assert.Contains(t, string(raw), resp.Header.Get("X-Request-ID"))
	})
}

func TestToolErrorStructuredContent(t *testing.T) {
	t.Parallel()
	ctx := withRequestID(context.Background(), "req-test")
	res := toolError(ctx, errors.New("postgres: password denied"))
	require.True(t, res.IsError)
	require.NotNil(t, res.StructuredContent)
	body, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok)
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "internal", errObj["code"])
	assert.Equal(t, "internal error", errObj["message"])
	assert.NotContains(t, errObj["message"], "postgres")
	trace, ok := body["trace"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "req-test", trace["request_id"])
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcpsdk.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, `"code":"internal"`)

	invalid := toolError(ctx, types.Errorf(types.ErrInvalidArgument, "name is required"))
	invalidBody, ok := invalid.StructuredContent.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "invalid_argument", invalidBody["error"].(map[string]any)["code"])
	require.NoError(t, pkgmcp.ValidateOutput(pkgmcp.OutputSchema("pipeline", "list"), invalidBody))
}
