package server

import (
	"strconv"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
)

func (c *Controller) hubApps(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(homelab.DefaultRegistry.List()))
}

func (c *Controller) hubApp(ctx fiber.Ctx) error {
	app, ok := homelab.DefaultRegistry.Get(ctx.Params("name"))
	if !ok {
		return types.Errorf(types.ErrNotFound, "app not found")
	}
	return ctx.JSON(protocol.NewSuccessResponse(app))
}

func (c *Controller) hubAppStatus(ctx fiber.Ctx) error {
	app, ok := homelab.DefaultRegistry.Get(ctx.Params("name"))
	if !ok {
		return types.Errorf(types.ErrNotFound, "app not found")
	}
	status, err := homelabRuntime.Status(ctx.Context(), app)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "status": status}))
}

func (c *Controller) hubAppLogs(ctx fiber.Ctx) error {
	app, ok := homelab.DefaultRegistry.Get(ctx.Params("name"))
	if !ok {
		return types.Errorf(types.ErrNotFound, "app not found")
	}
	tail := 100
	if raw := ctx.Query("tail"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			tail = parsed
		}
	}
	logs, err := homelabRuntime.Logs(ctx.Context(), app, tail)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "logs": logs}))
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
