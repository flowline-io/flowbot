package server

import (
	"bufio"
	"context"
	"errors"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

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

	// Capture the base context before streaming. SendStreamWriter runs its
	// callback in a separate goroutine after this handler returns, at which
	// point Fiber releases and reuses the fiber.Ctx. Touching c (e.g.
	// c.Context()) from inside the callback races with that release, so the
	// parent context must be resolved here. The deferred cancel below still
	// terminates the run when the stream ends (e.g. client disconnect).
	baseCtx := c.Context()

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

		runCtx, cancel := context.WithTimeout(baseCtx, chatagent.RunTimeout())
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
