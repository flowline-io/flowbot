package mcp

import (
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/config"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/version"
)

func registerServerCard(app *fiber.App) {
	if app == nil || !config.App.MCP.PublishServerCard() {
		return
	}
	app.Get("/mcp/server-card", serveServerCard)
}

func serveServerCard(c fiber.Ctx) error {
	endpoint := streamEndpoint(c)
	card, err := pkgmcp.NewServerCard(endpoint, version.Buildtags)
	if err != nil {
		return writeServerCardProblem(c, http.StatusBadRequest)
	}
	body, err := sonic.Marshal(card)
	if err != nil {
		return writeServerCardProblem(c, http.StatusInternalServerError)
	}
	c.Set(fiber.HeaderContentType, pkgmcp.ServerCardMediaType)
	return c.Send(body)
}

func streamEndpoint(c fiber.Ctx) string {
	origin := publicOrigin(c)
	if origin == "" {
		return ""
	}
	return origin + "/mcp"
}

func publicOrigin(c fiber.Ctx) string {
	if base := strings.TrimRight(strings.TrimSpace(config.App.Flowbot.URL), "/"); base != "" {
		return base
	}
	host := strings.TrimSpace(c.Host())
	if host == "" {
		return ""
	}
	scheme := "http"
	if strings.EqualFold(c.Get("X-Forwarded-Proto"), "https") || strings.EqualFold(c.Protocol(), "https") {
		scheme = "https"
	}
	return scheme + "://" + host
}

func writeServerCardProblem(c fiber.Ctx, status int) error {
	raw, err := sonic.Marshal(map[string]string{"error": "mcp endpoint is unavailable"})
	if err != nil {
		return err
	}
	c.Status(status)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(raw)
}
