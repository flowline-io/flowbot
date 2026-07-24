package server

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/route"
)

func (h *chatAgentHTTP) getPermissions(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID != "" {
		if err := h.ensureSessionOwner(c, sessionID); err != nil {
			return chatAgentError(c, err)
		}
	}
	view, err := h.service.BuildPermissionsView(c.Context(), rc.UID, sessionID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (h *chatAgentHTTP) putPermissions(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	cfg, err := chatagent.ParsePermissionsBody(c.Body())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := chatagent.SaveUserPermissions(c.Context(), rc.UID, cfg); err != nil {
		return chatAgentError(c, err)
	}
	view, err := h.service.BuildPermissionsView(c.Context(), rc.UID, "")
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (h *chatAgentHTTP) deletePermissions(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if err := chatagent.DeleteUserPermissions(c.Context(), rc.UID); err != nil {
		return chatAgentError(c, err)
	}
	view, err := h.service.BuildPermissionsView(c.Context(), rc.UID, "")
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (h *chatAgentHTTP) clearPermissionGrants(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	h.service.ClearSessionPermissionGrants(c.Context(), sessionID)
	return c.SendStatus(fiber.StatusNoContent)
}
