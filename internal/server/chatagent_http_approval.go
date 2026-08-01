package server

import (
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/route"
)

func (*chatAgentHTTP) getApproval(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	view, err := chatagent.BuildApprovalView(c.Context(), rc.UID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (*chatAgentHTTP) putApproval(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	mode, err := chatagent.ParseApprovalBody(c.Body())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := chatagent.SaveUserApprovalMode(c.Context(), rc.UID, mode); err != nil {
		return chatAgentError(c, err)
	}
	view, err := chatagent.BuildApprovalView(c.Context(), rc.UID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (*chatAgentHTTP) deleteApproval(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if err := chatagent.DeleteUserApprovalMode(c.Context(), rc.UID); err != nil {
		return chatAgentError(c, err)
	}
	view, err := chatagent.BuildApprovalView(c.Context(), rc.UID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}
