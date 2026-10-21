package mcp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/version"
)

func TestServerCard(t *testing.T) {
	prevMCP := config.App.MCP
	prevURL := config.App.Flowbot.URL
	t.Cleanup(func() {
		config.App.MCP = prevMCP
		config.App.Flowbot.URL = prevURL
	})

	t.Run("returns the public card for this host", func(t *testing.T) {
		config.App.MCP = mcpConfig(true, nil)
		config.App.Flowbot.URL = ""
		app := fiber.New()
		Register(app, nil)

		req := httptest.NewRequest(http.MethodGet, "http://flowbot.example:6060/mcp/server-card?x=1", http.NoBody)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, pkgmcp.ServerCardMediaType, resp.Header.Get("Content-Type"))
		assert.Empty(t, resp.Header.Get("ETag"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))

		var doc map[string]any
		require.NoError(t, sonic.Unmarshal(body, &doc))
		assert.Equal(t, []string{"$schema", "description", "name", "remotes", "version"}, sortedKeys(doc))
		assert.Equal(t, pkgmcp.ServerCardSchema, doc["$schema"])
		assert.Equal(t, "flowline-io/flowbot", doc["name"])
		assert.Equal(t, version.Buildtags, doc["version"])
		assert.NotContains(t, string(body), "tools")
		assert.NotContains(t, string(body), "scope")
		assert.NotContains(t, doc["description"], "flowbot.example")
		remotes, ok := doc["remotes"].([]any)
		require.True(t, ok)
		require.Len(t, remotes, 1)
		remote, ok := remotes[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "http://flowbot.example:6060/mcp", remote["url"])
		assert.Equal(t, []any{pkgmcp.SupportedProtocolVersion}, remote["supportedProtocolVersions"])
		assert.NotContains(t, remote, "headers")
	})

	t.Run("uses https when the proxy forwarded it", func(t *testing.T) {
		config.App.MCP = mcpConfig(true, nil)
		config.App.Flowbot.URL = ""
		app := fiber.New()
		Register(app, nil)
		req := httptest.NewRequest(http.MethodGet, "http://flowbot.example:6060/mcp/server-card", http.NoBody)
		req.Header.Set("X-Forwarded-Proto", "https")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var doc map[string]any
		require.NoError(t, sonic.Unmarshal(body, &doc))
		remotes, ok := doc["remotes"].([]any)
		require.True(t, ok)
		remote, ok := remotes[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://flowbot.example:6060/mcp", remote["url"])
	})

	t.Run("uses flowbot.url as the remote origin", func(t *testing.T) {
		config.App.MCP = mcpConfig(true, nil)
		config.App.Flowbot.URL = "https://bot.example.com/"
		app := fiber.New()
		Register(app, nil)
		req := httptest.NewRequest(http.MethodGet, "http://flowbot.example:6060/mcp/server-card", http.NoBody)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var doc map[string]any
		require.NoError(t, sonic.Unmarshal(body, &doc))
		remotes, ok := doc["remotes"].([]any)
		require.True(t, ok)
		remote, ok := remotes[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://bot.example.com/mcp", remote["url"])
		assert.NotContains(t, remote["url"], "flowbot.example")
	})

	t.Run("returns json when the endpoint cannot be built", func(t *testing.T) {
		config.App.MCP = mcpConfig(true, nil)
		config.App.Flowbot.URL = "not a url"
		app := fiber.New()
		Register(app, nil)
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "http://flowbot.example/mcp/server-card", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		assert.JSONEq(t, `{"error":"mcp endpoint is unavailable"}`, string(body))
	})

	t.Run("does not require a token", func(t *testing.T) {
		config.App.MCP = mcpConfig(true, new(true))
		config.App.Flowbot.URL = ""
		app := fiber.New()
		Register(app, nil)
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "http://flowbot.example/mcp/server-card", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("stays off when mcp is disabled", func(t *testing.T) {
		config.App.MCP = mcpConfig(false, new(true))
		app := fiber.New()
		Register(app, nil)
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "http://flowbot.example/mcp/server-card", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("can be closed while mcp stays mounted", func(t *testing.T) {
		config.App.MCP = mcpConfig(true, new(false))
		app := fiber.New()
		Register(app, nil)
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "http://flowbot.example/mcp/server-card", http.NoBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mcpResp, err := app.Test(httptest.NewRequest(http.MethodGet, "http://flowbot.example/mcp", http.NoBody))
		require.NoError(t, err)
		defer mcpResp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, mcpResp.StatusCode)
	})
}

func mcpConfig(enabled bool, card *bool) config.MCPConfig {
	return config.MCPConfig{Enabled: enabled, ServerCard: card}
}

func sortedKeys(doc map[string]any) []string {
	keys := make([]string, 0, len(doc))
	for key := range doc {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
