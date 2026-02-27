// admin.go is a thin adapter that wires the shared adminctl package into
// the main server's fx dependency graph.
package server

import (
	"github.com/flowline-io/flowbot/internal/admin"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/gofiber/fiber/v3"
)

// newAdminController creates an AdminController instance (injected via fx.Provide).
func newAdminController() *admin.AdminController {
	return admin.NewAdminController(admin.Options{
		SlackClientID: config.App.Platform.Slack.ClientID,
	})
}

// handleAdminRoutes registers Admin API routes on the Fiber app.
// The main server only provides API endpoints; PWA pages are served by cmd/app.
func handleAdminRoutes(a *fiber.App, ac *admin.AdminController) {
	admin.HandleAPIRoutes(a, ac)
}
