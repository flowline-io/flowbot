// admin.go is a thin adapter that wires the shared adminctl package into
// the main server's fx dependency graph.
package server

import (
	"encoding/json"
	"log"

	"github.com/flowline-io/flowbot/internal/admin"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	slackProvider "github.com/flowline-io/flowbot/pkg/providers/slack"
	"github.com/gofiber/fiber/v3"
)

// newAdminController creates an AdminController instance (injected via fx.Provide).
func newAdminController() *admin.AdminController {
	return admin.NewAdminController(admin.Options{
		SlackClientID:     config.App.Platform.Slack.ClientID,
		SlackClientSecret: config.App.Platform.Slack.ClientSecret,
		OAuthStore: func(uid, accessToken string, extra []byte) error {
			var extraJSON model.JSON
			if len(extra) > 0 {
				if err := json.Unmarshal(extra, &extraJSON); err != nil {
					extraJSON = model.JSON{}
				}
			}
			err := store.Database.OAuthSet(model.OAuth{
				UID:   uid,
				Topic: "",
				Name:  slackProvider.ID,
				Type:  slackProvider.ID,
				Token: accessToken,
				Extra: extraJSON,
			})
			if err != nil {
				log.Printf("failed to persist slack oauth token for uid=%s: %v", uid, err)
				return err
			}
			log.Printf("slack oauth token persisted for uid=%s", uid)
			return nil
		},
	})
}

// handleAdminRoutes registers Admin API routes on the Fiber app.
// The main server only provides API endpoints; PWA pages are served by cmd/app.
func handleAdminRoutes(a *fiber.App, ac *admin.AdminController) {
	admin.HandleAPIRoutes(a, ac)
}
