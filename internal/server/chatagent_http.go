package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RegisterChatAgentRoutes wires Chat Agent REST endpoints for the terminal client.
func RegisterChatAgentRoutes(a *fiber.App) {
	chatHTTP := newChatAgentHTTP()
	a.Get("/chatagent/info", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.info)))
	a.Get("/chatagent/sessions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listSessions)))
	a.Post("/chatagent/sessions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.createSession)))
	a.Delete("/chatagent/sessions/:id", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.closeSession)))
	a.Get("/chatagent/sessions/:id/messages", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listMessages)))
	a.Get("/chatagent/sessions/:id/export", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.exportSession)))
	a.Get("/chatagent/sessions/:id/context", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.contextUsage)))
	a.Post("/chatagent/sessions/:id/compact", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.compactSession)))
	a.Post("/chatagent/sessions/:id/messages", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.sendMessage)))
	a.Post("/chatagent/sessions/:id/confirm", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.confirm)))
	a.Post("/chatagent/sessions/:id/cancel", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.cancelRun)))
	a.Get("/chatagent/permissions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.getPermissions)))
	a.Put("/chatagent/permissions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.putPermissions)))
	a.Delete("/chatagent/permissions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.deletePermissions)))
	a.Get("/chatagent/sessions/:id/events", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.sessionEvents)))
	a.Delete("/chatagent/sessions/:id/permission-grants", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.clearPermissionGrants)))
}

type chatAgentHTTP struct {
	service *chatagent.Service
}

func newChatAgentHTTP() *chatAgentHTTP {
	return &chatAgentHTTP{service: chatagent.NewService()}
}

func (*chatAgentHTTP) info(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	info, err := chatagent.BuildAgentInfo(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(info)
}

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

type sendMessageBody struct {
	Text string `json:"text"`
}

func (h *chatAgentHTTP) sendMessage(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}

	var body sendMessageBody
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	if strings.TrimSpace(body.Text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "empty message"})
	}
	if _, ok := chatagent.GetAPIRunState(sessionID); ok {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": chatagent.ErrRunInFlight.Error()})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		hub := chatagent.GetSessionEventHub(sessionID)
		subID := "run"
		publisher := hub.Subscribe(subID, 64)
		defer hub.Unsubscribe(subID)

		gate := chatagent.NewConfirmGate(sessionID, nil)
		runState := chatagent.NewAPIRunState(publisher, gate)
		if err := chatagent.TrySetAPIRunState(sessionID, runState); err != nil {
			_ = writeChatAgentSSE(w, chatagent.StreamEvent{
				Type:    chatagent.EventTypeError,
				Message: err.Error(),
			})
			return
		}
		defer chatagent.ClearAPIRunState(sessionID, runState)

		runCtx, cancel := context.WithTimeout(c.Context(), chatagent.RunTimeout())
		defer cancel()
		chatagent.BindRunCancel(sessionID, cancel)
		defer chatagent.UnbindRunCancel(sessionID)

		runDone := make(chan error, 1)
		go func() {
			runDone <- h.service.RunAPI(runCtx, chatagent.RunRequest{
				SessionID: sessionID,
				Text:      body.Text,
			}, &chatagent.APIRunOptions{
				Publisher: publisher,
				Confirm:   gate,
			})
			publisher.Close()
		}()

		for {
			select {
			case ev, ok := <-publisher.Events():
				if !ok {
					return
				}
				if writeChatAgentSSE(w, ev) {
					return
				}
			case err := <-runDone:
				drainChatAgentSSE(w, publisher)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						_ = writeChatAgentSSE(w, chatagent.StreamEvent{
							Type:    chatagent.EventTypeCanceled,
							Message: "run canceled by user",
						})
						return
					}
					_ = writeChatAgentSSE(w, chatagent.StreamEvent{
						Type:    chatagent.EventTypeError,
						Message: err.Error(),
					})
				}
				return
			}
		}
	})
}

type confirmBody struct {
	ID       string `json:"id"`
	Approved bool   `json:"approved"`
	Mode     string `json:"mode"`
	Pattern  string `json:"pattern"`
}

func (h *chatAgentHTTP) confirm(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	var body confirmBody
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	reason := chatagent.ConfirmReasonDenied
	if body.Approved {
		reason = chatagent.ConfirmReasonApproved
	}
	mode := chatagent.ConfirmMode(body.Mode)
	if mode == "" {
		if body.Approved {
			mode = chatagent.ConfirmModeOnce
		} else {
			mode = chatagent.ConfirmModeReject
		}
	}
	ok, err := chatagent.ResolveConfirm(sessionID, body.ID, body.Approved, mode, body.Pattern, reason)
	if errors.Is(err, chatagent.ErrConfirmNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, chatagent.ErrConfirmResolved) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if !ok {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "confirm not applied"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *chatAgentHTTP) cancelRun(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	chatagent.CancelSessionRun(sessionID)
	if state, ok := chatagent.GetAPIRunState(sessionID); ok {
		if pub := state.Publisher(); pub != nil {
			_ = pub.Publish(chatagent.StreamEvent{
				Type:    chatagent.EventTypeCanceled,
				Message: "run canceled by user",
			})
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (*chatAgentHTTP) ensureSessionOwner(c fiber.Ctx, sessionID string) error {
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return types.ErrUnauthorized
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	sess, err := store.Database.GetChatSession(c.Context(), sessionID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.ErrNotFound
		}
		return err
	}
	if sess.UID != rc.UID.String() {
		return types.ErrForbidden
	}
	if sess.State == int(schema.ChatSessionClosed) {
		return types.ErrNotFound
	}
	return nil
}

func requireChatAgentEnabled() error {
	if !config.ChatAgentEnabled() {
		return chatagent.ErrChatAgentDisabled
	}
	return nil
}

func chatAgentError(c fiber.Ctx, err error) error {
	if errors.Is(err, chatagent.ErrChatAgentDisabled) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, chatagent.ErrRunInFlight) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if status, ok := domainErrorStatus(err); ok {
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	flog.Warn("[chat-agent] http error: %v", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

func writeChatAgentSSE(w *bufio.Writer, event chatagent.StreamEvent) bool {
	frame, err := chatagent.FormatSSEData(event)
	if err != nil {
		return true
	}
	if _, err := fmt.Fprint(w, frame); err != nil {
		return true
	}
	if err := w.Flush(); err != nil {
		return true
	}
	return event.Type == chatagent.EventTypeDone ||
		event.Type == chatagent.EventTypeError ||
		event.Type == chatagent.EventTypeCanceled
}

func drainChatAgentSSE(w *bufio.Writer, publisher *chatagent.ChannelPublisher) {
	for {
		select {
		case ev, ok := <-publisher.Events():
			if !ok {
				return
			}
			if writeChatAgentSSE(w, ev) {
				return
			}
		default:
			return
		}
	}
}

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
	view, err := chatagent.BuildPermissionsView(c.Context(), rc.UID, sessionID)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (*chatAgentHTTP) putPermissions(c fiber.Ctx) error {
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
	view, err := chatagent.BuildPermissionsView(c.Context(), rc.UID, "")
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(view)
}

func (*chatAgentHTTP) deletePermissions(c fiber.Ctx) error {
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
	view, err := chatagent.BuildPermissionsView(c.Context(), rc.UID, "")
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
	chatagent.ClearSessionPermissionGrants(sessionID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *chatAgentHTTP) sessionEvents(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := c.Params("id")
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		hub := chatagent.GetSessionEventHub(sessionID)
		subID := fmt.Sprintf("observer-%p", w)
		publisher := hub.Subscribe(subID, 32)
		defer hub.Unsubscribe(subID)

		ctx := c.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-publisher.Events():
				if !ok {
					return
				}
				switch ev.Type {
				case chatagent.EventTypeConfirm, chatagent.EventTypeConfirmResolved, chatagent.EventTypeCanceled:
					if writeChatAgentSSE(w, ev) {
						return
					}
				}
			}
		}
	})
}
