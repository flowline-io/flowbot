package server

import (
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
)

func (c *Controller) hubApps(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse([]any{}))
}

func (c *Controller) hubApp(ctx fiber.Ctx) error {
	return protocol.ErrNotFound.New("homelab app registry is not initialized")
}

func (c *Controller) hubCapabilities(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(hub.Default.List()))
}

func (c *Controller) hubCapability(ctx fiber.Ctx) error {
	capabilityType := hub.CapabilityType(ctx.Params("type"))
	desc, ok := hub.Default.Get(capabilityType)
	if !ok {
		return protocol.ErrNotFound.New("capability not found")
	}
	return ctx.JSON(protocol.NewSuccessResponse(desc))
}

func (c *Controller) hubHealth(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{
		"status":       "ok",
		"capabilities": hub.Default.List(),
	}))
}
