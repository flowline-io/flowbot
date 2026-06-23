package server

import (
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/route"
)

func (*chatAgentHTTP) listScheduledTasks(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	tasks, err := chatagent.ListScheduledTasksForUID(c.Context(), rc.UID, nil)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{"tasks": tasks})
}

func (*chatAgentHTTP) createScheduledTask(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body chatagent.CreateScheduledTaskRequest
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	if err := chatagent.ParseCreateScheduledTaskRequest(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	sourceSessionID := strings.TrimSpace(c.Query("source_session_id"))
	task, err := chatagent.CreateScheduledTaskForUID(c.Context(), rc.UID, sourceSessionID, body)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(task)
}

func (*chatAgentHTTP) getScheduledTask(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	taskID := c.Params("id")
	task, err := chatagent.GetScheduledTaskForUID(c.Context(), rc.UID, taskID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(task)
}

func (*chatAgentHTTP) patchScheduledTask(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	taskID := c.Params("id")
	var body chatagent.UpdateScheduledTaskRequest
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	task, err := chatagent.PatchScheduledTaskForUID(c.Context(), rc.UID, taskID, body)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(task)
}

func (*chatAgentHTTP) cancelScheduledTask(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	taskID := c.Params("id")
	if err := chatagent.CancelScheduledTaskForUID(c.Context(), rc.UID, taskID); err != nil {
		return chatAgentError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (*chatAgentHTTP) listScheduledTaskRuns(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	taskID := c.Params("id")
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid limit"})
		}
		limit = parsed
	}
	runs, err := chatagent.ListScheduledTaskRuns(c.Context(), rc.UID, taskID, limit)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{"runs": runs})
}
