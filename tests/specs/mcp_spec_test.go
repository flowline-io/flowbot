//go:build integration

package specs

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/oops"

	serversmcp "github.com/flowline-io/flowbot/internal/server/mcp"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

var _ = Describe("MCP /mcp", Label("mcp"), func() {
	var (
		app        *fiber.App
		prevEnable bool
		prevCard   *bool
		apiToken   string
		fullToken  string
	)

	BeforeEach(func() {
		prevEnable = config.App.MCP.Enabled
		prevCard = config.App.MCP.ServerCard
		config.App.MCP.Enabled = true
		config.App.MCP.ServerCard = nil
		app = fiber.New(fiber.Config{
			ErrorHandler: func(ctx fiber.Ctx, err error) error {
				if err == nil {
					return nil
				}
				if errors.Is(err, types.ErrForbidden) {
					return ctx.Status(fiber.StatusForbidden).JSON(protocol.NewFailedResponse(err))
				}
				var fiberErr *fiber.Error
				if errors.As(err, &fiberErr) {
					return ctx.Status(fiberErr.Code).JSON(protocol.NewFailedResponse(err))
				}
				var e oops.OopsError
				if errors.As(err, &e) && e.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return ctx.Status(fiber.StatusUnauthorized).JSON(protocol.NewFailedResponse(e))
				}
				return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(err))
			},
		})
		app.Use(requestid.New())
		serversmcp.SetRateRedis(Redis)
		serversmcp.Register(app, nil)

		uid := "bdd-mcp-user-" + types.Id()
		apiToken = "bdd-mcp-api-" + types.Id()
		fullToken = "bdd-mcp-full-" + types.Id()
		mds := store.NewModuleDataStore(EntClient)
		Expect(mds.ParameterSet(context.Background(), auth.HashToken(apiToken), types.KV{
			"uid": uid, "topic": "cli", "kind": "api",
			"scopes": []string{auth.ScopePipelineRun, auth.ScopeHubAppsRead, auth.ScopeHubHealthRead},
		}, time.Now().Add(time.Hour))).To(Succeed())
		Expect(mds.ParameterSet(context.Background(), auth.HashToken(fullToken), types.KV{
			"uid": uid, "topic": "web", "kind": webauth.KindFull,
			"scopes": []string{auth.ScopeAdmin},
		}, time.Now().Add(time.Hour))).To(Succeed())
	})

	AfterEach(func() {
		serversmcp.SetRateRedis(nil)
		config.App.MCP.Enabled = prevEnable
		config.App.MCP.ServerCard = prevCard
	})

	It("returns the public server card without a token", func() {
		prevOrigin := config.App.Flowbot.URL
		config.App.Flowbot.URL = ""
		DeferCleanup(func() { config.App.Flowbot.URL = prevOrigin })
		resp, err := app.Test(MakeRequest(http.MethodGet, "/mcp/server-card", nil))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(Equal("application/mcp-server-card+json"))
		var doc map[string]any
		Expect(sonic.Unmarshal(ReadBody(resp), &doc)).To(Succeed())
		Expect(doc["name"]).To(Equal("flowline-io/flowbot"))
		Expect(doc).NotTo(HaveKey("tools"))
		remotes, ok := doc["remotes"].([]any)
		Expect(ok).To(BeTrue())
		Expect(remotes).To(HaveLen(1))
		remote, ok := remotes[0].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(remote["url"]).To(Equal("http://localhost/mcp"))
		Expect(remote["supportedProtocolVersions"]).To(Equal([]any{"2026-07-28"}))
	})

	It("hides the server card when mcp.server_card is false", func() {
		off := false
		config.App.MCP.ServerCard = &off
		hidden := fiber.New()
		serversmcp.Register(hidden, nil)
		resp, err := hidden.Test(MakeRequest(http.MethodGet, "/mcp/server-card", nil))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("returns 405 for GET without a token", func() {
		resp, err := app.Test(MakeRequest(http.MethodGet, "/mcp", nil))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
	})

	It("returns 405 for DELETE without a token", func() {
		resp, err := app.Test(MakeRequest(http.MethodDelete, "/mcp", nil))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
	})

	It("returns 401 for POST without a token", func() {
		resp, err := app.Test(JSONRequest(http.MethodPost, "/mcp", []byte(`{}`)))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("returns 403 for POST with a kind=full web session", func() {
		req := JSONRequest(http.MethodPost, "/mcp", []byte(`{}`))
		req.Header.Set("Authorization", "Bearer "+fullToken)
		resp, err := app.Test(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	})

	It("accepts a scoped API token on POST", func() {
		req := JSONRequest(http.MethodPost, "/mcp", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
		req.Header.Set("Authorization", "Bearer "+apiToken)
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).NotTo(Equal(http.StatusUnauthorized))
		Expect(resp.StatusCode).NotTo(Equal(http.StatusForbidden))
		Expect(resp.StatusCode).NotTo(Equal(http.StatusMethodNotAllowed))
	})

	It("returns structuredContent.error when pipeline.get has no name", func() {
		resp := postMCP(app, apiToken, mcpCallBody("pipeline.get", map[string]any{}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		result := mcpResult(body)
		Expect(result["isError"]).To(BeTrue())
		sc := structuredContent(result)
		errObj, ok := sc["error"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(errObj["code"]).To(Equal("invalid_argument"))
		Expect(errObj["field"]).To(Equal("name"))
		Expect(errObj["hint"]).To(Equal("Fix field name."))
		Expect(errObj["retryable"]).To(BeFalse())
		requestID := resp.Header.Get("X-Request-ID")
		Expect(requestID).NotTo(BeEmpty())
		trace, ok := sc["trace"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(trace["request_id"]).To(Equal(requestID))
		Expect(textJSON(result)).To(Equal(sc))
	})

	It("returns items and trace.request_id for hub.apps", func() {
		resp := postMCP(app, apiToken, mcpCallBody("hub.apps", map[string]any{}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		result := mcpResult(body)
		expectToolSucceeded(result)
		sc := structuredContent(result)
		Expect(sc["items"]).To(BeAssignableToTypeOf([]any{}))
		requestID := resp.Header.Get("X-Request-ID")
		Expect(requestID).NotTo(BeEmpty())
		trace, ok := sc["trace"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(trace["request_id"]).To(Equal(requestID))
		Expect(textJSON(result)).To(Equal(sc))
	})

	It("returns permission_denied when a read token calls pipeline.run", func() {
		readToken := "bdd-mcp-read-" + types.Id()
		uid := "bdd-mcp-read-user-" + types.Id()
		mds := store.NewModuleDataStore(EntClient)
		Expect(mds.ParameterSet(context.Background(), auth.HashToken(readToken), types.KV{
			"uid": uid, "topic": "cli", "kind": "api",
			"scopes": []string{auth.ScopePipelineRead, auth.ScopeHubAppsRead},
		}, time.Now().Add(time.Hour))).To(Succeed())
		resp := postMCP(app, readToken, mcpCallBody("pipeline.run", map[string]any{"name": "daily"}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		result := mcpResult(body)
		Expect(result["isError"]).To(BeTrue())
		errObj, ok := structuredContent(result)["error"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(errObj["code"]).To(Equal("permission_denied"))
	})

	It("uses limit 20 when hub.apps omits limit", func() {
		resp := postMCP(app, apiToken, mcpCallBody("hub.apps", map[string]any{}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		page, ok := structuredContent(mcpResult(body))["page"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(page["limit"]).To(Equal(float64(20)))
	})

	It("clamps hub.apps limit above 100 to 100", func() {
		resp := postMCP(app, apiToken, mcpCallBody("hub.apps", map[string]any{"limit": 500}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		page, ok := structuredContent(mcpResult(body))["page"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(page["limit"]).To(Equal(float64(100)))
	})

	It("sets page.truncated when hub.apps exceeds mcp.max_output_bytes", func() {
		prev := config.App.MCP.MaxOutputBytes
		config.App.MCP.MaxOutputBytes = 1
		defer func() { config.App.MCP.MaxOutputBytes = prev }()
		resp := postMCP(app, apiToken, mcpCallBody("hub.apps", map[string]any{"limit": 20}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		page, ok := structuredContent(mcpResult(body))["page"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(page["truncated"]).To(BeTrue())
	})

	It("returns an object and trace.request_id for hub.health", func() {
		resp := postMCP(app, apiToken, mcpCallBody("hub.health", map[string]any{}))
		body := ReadBody(resp)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		result := mcpResult(body)
		expectToolSucceeded(result)
		sc := structuredContent(result)
		Expect(sc["status"]).NotTo(BeEmpty())
		Expect(sc).NotTo(HaveKey("items"))
		requestID := resp.Header.Get("X-Request-ID")
		Expect(requestID).NotTo(BeEmpty())
		trace, ok := sc["trace"].(map[string]any)
		Expect(ok).To(BeTrue(), string(body))
		Expect(trace["request_id"]).To(Equal(requestID))
		Expect(textJSON(result)).To(Equal(sc))
	})
})

func postMCP(app *fiber.App, token string, body []byte) *http.Response {
	req := JSONRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json, text/event-stream")
	// Protocol 2026-07-28: the header must match _meta, and tools/call also needs method and name.
	req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	var msg struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := sonic.Unmarshal(body, &msg); err == nil {
		if msg.Method != "" {
			req.Header.Set("Mcp-Method", msg.Method)
		}
		if msg.Params.Name != "" {
			req.Header.Set("Mcp-Name", msg.Params.Name)
		}
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	Expect(err).NotTo(HaveOccurred())
	return resp
}

func mcpCallBody(name string, args map[string]any) []byte {
	raw, err := sonic.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": args,
			"_meta": map[string]any{
				"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
				"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	return raw
}

func mcpResult(body []byte) map[string]any {
	raw := bytes.TrimSpace(body)
	if bytes.HasPrefix(raw, []byte("event:")) || bytes.Contains(raw, []byte("\ndata:")) {
		for _, line := range bytes.Split(raw, []byte("\n")) {
			line = bytes.TrimSpace(line)
			if bytes.HasPrefix(line, []byte("data:")) {
				raw = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
				break
			}
		}
	}
	var msg map[string]any
	Expect(sonic.Unmarshal(raw, &msg)).To(Succeed(), string(body))
	result, ok := msg["result"].(map[string]any)
	Expect(ok).To(BeTrue(), string(body))
	return result
}

func expectToolSucceeded(result map[string]any) {
	GinkgoHelper()
	// CallToolResult omits isError when the call succeeded.
	if v, ok := result["isError"]; ok {
		Expect(v).To(Equal(false))
	}
}

func structuredContent(result map[string]any) map[string]any {
	sc, ok := result["structuredContent"].(map[string]any)
	Expect(ok).To(BeTrue())
	return sc
}

func textJSON(result map[string]any) map[string]any {
	content, ok := result["content"].([]any)
	Expect(ok).To(BeTrue())
	Expect(content).NotTo(BeEmpty())
	first, ok := content[0].(map[string]any)
	Expect(ok).To(BeTrue())
	text, ok := first["text"].(string)
	Expect(ok).To(BeTrue())
	var parsed map[string]any
	Expect(sonic.Unmarshal([]byte(text), &parsed)).To(Succeed())
	return parsed
}
