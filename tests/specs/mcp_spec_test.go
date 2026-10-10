//go:build integration

package specs

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
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
		apiToken   string
		fullToken  string
	)

	BeforeEach(func() {
		prevEnable = config.App.MCP.Enabled
		config.App.MCP.Enabled = true
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
		serversmcp.Register(app, nil)

		uid := "bdd-mcp-user-" + types.Id()
		apiToken = "bdd-mcp-api-" + types.Id()
		fullToken = "bdd-mcp-full-" + types.Id()
		mds := store.NewModuleDataStore(EntClient)
		Expect(mds.ParameterSet(context.Background(), auth.HashToken(apiToken), types.KV{
			"uid": uid, "topic": "cli", "kind": "api",
			"scopes": []string{auth.ScopePipelineRun},
		}, time.Now().Add(time.Hour))).To(Succeed())
		Expect(mds.ParameterSet(context.Background(), auth.HashToken(fullToken), types.KV{
			"uid": uid, "topic": "web", "kind": webauth.KindFull,
			"scopes": []string{auth.ScopeAdmin},
		}, time.Now().Add(time.Hour))).To(Succeed())
	})

	AfterEach(func() {
		config.App.MCP.Enabled = prevEnable
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
})
