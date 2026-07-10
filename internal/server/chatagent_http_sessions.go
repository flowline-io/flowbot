package server

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

func (*chatAgentHTTP) createSession(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	sessionID := types.Id()
	if err := chatagent.CreateSession(c.Context(), rc.UID, sessionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"session_id": sessionID})
}

func (*chatAgentHTTP) listSessions(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid limit"})
		}
		limit = parsed
	}
	sessions, nextCursor, err := chatagent.ListUserActiveSessions(c.Context(), rc.UID, limit, c.Query("cursor"))
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{
		"sessions": sessions,
		"cursor":   nextCursor,
	})
}

func (h *chatAgentHTTP) closeSession(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	if err := chatagent.CloseSession(c.Context(), sessionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	chatagent.ClearAPIRunState(sessionID, nil)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *chatAgentHTTP) listMessages(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	messages, err := chatagent.ListSessionMessages(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"messages": messages})
}

func (h *chatAgentHTTP) exportSession(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	export, err := chatagent.ExportSession(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(export)
}

func (h *chatAgentHTTP) contextUsage(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	report, err := chatagent.BuildContextUsageReport(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func (h *chatAgentHTTP) compactSession(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	result, err := h.service.CompactSession(c.Context(), sessionID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{
		"compacted":     result.Compacted,
		"tokens_before": result.TokensBefore,
		"tokens_after":  result.TokensAfter,
	})
}

func (h *chatAgentHTTP) getSessionMode(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{
		"mode":  chatagent.LoadSessionMode(c.Context(), sessionID),
		"title": chatagent.LoadSessionTitle(c.Context(), sessionID),
	})
}

type sessionModeBody struct {
	Mode string `json:"mode"`
}

func (h *chatAgentHTTP) putSessionMode(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	var body sessionModeBody
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	mode := strings.TrimSpace(body.Mode)
	if !chatagent.ValidSessionMode(mode) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid mode"})
	}
	if err := chatagent.SetSessionModeAndNotify(c.Context(), sessionID, mode); err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(fiber.Map{"mode": mode})
}

func (h *chatAgentHTTP) sessionEvents(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := strings.Clone(c.Params("id"))
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Resolve the request context before streaming. The SendStreamWriter
	// callback runs in a separate goroutine after this handler returns, when
	// Fiber has released and reused the fiber.Ctx; calling c.Context() from
	// inside the callback races with that release.
	reqCtx := c.Context()
	return c.SendStreamWriter(func(w *bufio.Writer) {
		hub := chatagent.GetSessionEventHub(sessionID)
		subID := fmt.Sprintf("observer-%p", w)
		publisher := hub.Subscribe(subID, 32)
		defer hub.Unsubscribe(subID)

		for {
			select {
			case <-reqCtx.Done():
				return
			case ev, ok := <-publisher.Events():
				if !ok {
					return
				}
				switch ev.Type {
				case chatagent.EventTypeConfirm, chatagent.EventTypeConfirmResolved, chatagent.EventTypeCanceled, chatagent.EventTypeModeChange:
					if writeChatAgentSSE(w, ev) {
						return
					}
				}
			}
		}
	})
}
