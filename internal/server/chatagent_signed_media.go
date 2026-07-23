package server

import (
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/media"
)

// RegisterChatAgentSignedMediaRoute serves HMAC-signed FS media for LLM providers.
func RegisterChatAgentSignedMediaRoute(a *fiber.App) {
	a.Get(media.SignedPathPrefix+":fileID", serveSignedChatAgentMedia)
}

func serveSignedChatAgentMedia(c fiber.Ctx) error {
	fileID := strings.TrimSpace(c.Params("fileID"))
	if fileID == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}
	secret := config.App.ChatAgent.Media.SignSecret
	if secret == "" && config.App.Media != nil {
		secret = config.App.Media.SignSecret
	}
	if err := media.VerifySignedRequest(secret, fileID, c.Query("exp"), c.Query("sig"), time.Now().UTC()); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	accessor, ok := media.AsAccessor(store.FileSystem)
	if !ok || accessor == nil {
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	fd, rc, err := accessor.OpenByID(c.Context(), fileID)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	defer func() { _ = rc.Close() }()
	if fd != nil && fd.MimeType != "" {
		c.Set("Content-Type", fd.MimeType)
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	return c.Send(data)
}
