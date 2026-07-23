package server

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/route"
)

type sendMessageBody struct {
	Text        string                    `json:"text"`
	Attachments []chatagent.AttachmentRef `json:"attachments"`
}

func (h *chatAgentHTTP) sendMessage(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	// Clone before SendStreamWriter: Fiber recycles fasthttp buffers after the
	// handler returns; concurrent requests can overwrite Params("id") in place.
	sessionID := strings.Clone(c.Params("id"))
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}

	var body sendMessageBody
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	text := strings.Clone(strings.TrimSpace(body.Text))
	attachments := append([]chatagent.AttachmentRef(nil), body.Attachments...)
	if text == "" && len(attachments) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "empty message"})
	}
	if _, ok := chatagent.GetAPIRunState(sessionID); ok {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": chatagent.ErrRunInFlight.Error()})
	}

	ownerUID := chatAgentCallerUID(c)

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
		sse := &chatagent.BufioSSEWriter{W: w}
		chatagent.StreamAPIRun(baseCtx, h.service, sessionID, text, attachments, ownerUID, sse)
	})
}

func (h *chatAgentHTTP) uploadSessionMedia(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	sessionID := strings.Clone(c.Params("id"))
	if err := h.ensureSessionOwner(c, sessionID); err != nil {
		return chatAgentError(c, err)
	}
	if err := chatagent.EnsureMediaPublicConfig(); err != nil {
		return chatAgentError(c, err)
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "open file failed"})
	}
	defer func() { _ = file.Close() }()

	var seeker io.ReadSeeker
	if rs, ok := file.(io.ReadSeeker); ok {
		seeker = rs
	} else {
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "read file failed"})
		}
		seeker = bytes.NewReader(data)
	}
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	result, err := chatagent.UploadSessionMedia(c.Context(), sessionID, chatAgentCallerUID(c), fileHeader.Filename, mimeType, seeker, fileHeader.Size)
	if err != nil {
		return chatAgentError(c, err)
	}
	return c.JSON(result)
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
	sessionID := strings.Clone(c.Params("id"))
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
	sessionID := strings.Clone(c.Params("id"))
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

func chatAgentCallerUID(c fiber.Ctx) string {
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return ""
	}
	return rc.UID.String()
}
