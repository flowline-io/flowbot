package server

import (
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

func (h *chatAgentHTTP) getResource(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Query("session_id")
	if sessionID == "" {
		sessionID = c.Params("id")
	}
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "session_id is required"})
	}
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	uri := c.Query("uri")
	if uri == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "uri is required"})
	}
	content, err := chatagent.ResolveResourceWithOptions(c.Context(), sessionID, uri, chatagent.ResolveResourceOptions{
		Full: c.Query("full") == "1",
	})
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{
		"uri":          content.URI,
		"kind":         content.Kind,
		"title":        content.Title,
		"content":      content.Content,
		"content_type": content.ContentType,
		"truncated":    content.Truncated,
	})
}

func (h *chatAgentHTTP) listSessionPlans(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	plans, err := chatagent.ListPlanSummaries(c.Context(), sessionID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{"plans": plans})
}

func (h *chatAgentHTTP) listSessionTodos(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	todos, err := chatagent.ListTodoItems(c.Context(), sessionID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{"todos": todos})
}
