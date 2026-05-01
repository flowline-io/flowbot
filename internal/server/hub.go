package server

import (
	"strconv"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/route"
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

func (c *Controller) hubAppStart(ctx fiber.Ctx) error {
	app, err := c.requireAppWithLifecycleCheck(ctx, "start", auth.ScopeHubAppsStart)
	if err != nil {
		return err
	}
	if err := homelabRuntime.Start(ctx.Context(), app); err != nil {
		c.writeLifecycleAudit(ctx, app.Name, "hub.apps.start", "failed", err.Error())
		return err
	}
	c.writeLifecycleAudit(ctx, app.Name, "hub.apps.start", "success", "")
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "status": "started"}))
}

func (c *Controller) hubAppStop(ctx fiber.Ctx) error {
	app, err := c.requireAppWithLifecycleCheck(ctx, "stop", auth.ScopeHubAppsStop)
	if err != nil {
		return err
	}
	if err := homelabRuntime.Stop(ctx.Context(), app); err != nil {
		c.writeLifecycleAudit(ctx, app.Name, "hub.apps.stop", "failed", err.Error())
		return err
	}
	c.writeLifecycleAudit(ctx, app.Name, "hub.apps.stop", "success", "")
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "status": "stopped"}))
}

func (c *Controller) hubAppRestart(ctx fiber.Ctx) error {
	app, err := c.requireAppWithLifecycleCheck(ctx, "restart", auth.ScopeHubAppsRestart)
	if err != nil {
		return err
	}
	if err := homelabRuntime.Restart(ctx.Context(), app); err != nil {
		c.writeLifecycleAudit(ctx, app.Name, "hub.apps.restart", "failed", err.Error())
		return err
	}
	c.writeLifecycleAudit(ctx, app.Name, "hub.apps.restart", "success", "")
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "status": "restarted"}))
}

func (c *Controller) hubAppPull(ctx fiber.Ctx) error {
	app, err := c.requireAppWithLifecycleCheck(ctx, "pull", auth.ScopeHubAppsPull)
	if err != nil {
		return err
	}
	if err := homelabRuntime.Pull(ctx.Context(), app); err != nil {
		c.writeLifecycleAudit(ctx, app.Name, "hub.apps.pull", "failed", err.Error())
		return err
	}
	c.writeLifecycleAudit(ctx, app.Name, "hub.apps.pull", "success", "")
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "status": "pulled"}))
}

func (c *Controller) hubAppUpdate(ctx fiber.Ctx) error {
	app, err := c.requireAppWithLifecycleCheck(ctx, "update", auth.ScopeHubAppsUpdate)
	if err != nil {
		return err
	}
	if err := homelabRuntime.Update(ctx.Context(), app); err != nil {
		c.writeLifecycleAudit(ctx, app.Name, "hub.apps.update", "failed", err.Error())
		return err
	}
	c.writeLifecycleAudit(ctx, app.Name, "hub.apps.update", "success", "")
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"name": app.Name, "status": "updated"}))
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
	checker := hub.NewChecker(hub.Default)
	result := checker.Check(ctx.Context())
	if result.Status == hub.HealthUnhealthy {
		return ctx.JSON(protocol.NewFailedResponse(types.Errorf(types.ErrUnavailable, "hub unhealthy")))
	}
	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func (c *Controller) requireAppWithLifecycleCheck(ctx fiber.Ctx, operation string, scope string) (homelab.App, error) {
	name := ctx.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return app, types.Errorf(types.ErrNotFound, "app not found")
	}
	if !route.ScopeHandler(ctx, scope) {
		c.writeLifecycleAudit(ctx, name, "hub.apps."+operation, "rejected", "insufficient scope: "+scope)
		return app, types.Errorf(types.ErrForbidden, "insufficient scope: %s", scope)
	}
	perm := homelab.DefaultRegistry.Permissions()
	if !checkLifecyclePermission(perm, operation) {
		c.writeLifecycleAudit(ctx, name, "hub.apps."+operation, "rejected", "config permission denied")
		return app, types.Errorf(types.ErrForbidden, "%s not allowed by config for app %s", operation, name)
	}
	return app, nil
}

func (c *Controller) writeLifecycleAudit(ctx fiber.Ctx, appName, action, result, errMsg string) {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	auditStore := store.NewAuditStore(store.Database.GetDB())
	_ = auditStore.Write(store.AuditEntry{
		ActorType:    "token",
		ActorID:      uid.String(),
		UID:          uid.String(),
		Topic:        topic,
		Action:       action,
		ResourceType: "app",
		ResourceName: appName,
		Result:       result,
		Error:        errMsg,
		IPAddress:    ctx.IP(),
		UserAgent:    ctx.Get("User-Agent"),
	})
}

func checkLifecyclePermission(perm homelab.Permissions, operation string) bool {
	switch operation {
	case "status":
		return perm.Status
	case "logs":
		return perm.Logs
	case "start":
		return perm.Start
	case "stop":
		return perm.Stop
	case "restart":
		return perm.Restart
	case "pull":
		return perm.Pull
	case "update":
		return perm.Update
	default:
		return false
	}
}
