package server

import (
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/version"
)

// versionHandler returns the running server's build version (public, no auth).
func versionHandler(c fiber.Ctx) error {
	return c.JSON(protocol.NewSuccessResponse(types.KV{
		"version":    version.Buildtags,
		"buildstamp": version.Buildstamp,
	}))
}
