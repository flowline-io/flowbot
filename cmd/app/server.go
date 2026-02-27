//go:build !js

package main

import (
	"context"
	"log"

	"github.com/flowline-io/flowbot/cmd/app/config"
	"github.com/flowline-io/flowbot/internal/admin"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

// main is the native (server) entry point.
// It uses fx for dependency injection and lifecycle management.
func main() {
	fx.New(
		fx.Provide(config.NewConfig),
		fx.Invoke(startServer),
	).Run()
}

// startServer creates a Fiber app with admin API routes and the PWA handler,
// then starts listening on the configured address.
func startServer(lc fx.Lifecycle, cfg config.Type) {
	registerRoutes()

	app := fiber.New()

	// Register PWA page routes (API is served by the main server).
	// Pass the API base URL so the Wasm client knows where to send requests.
	apiBaseURL := cfg.API.URL + cfg.API.Prefix
	admin.HandlePageRoutes(app, apiBaseURL)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			log.Printf("Admin PWA server listening on %s", cfg.Listen)
			go func() {
				if err := app.Listen(cfg.Listen, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
					log.Fatalf("Fiber server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			log.Println("Shutting down Admin PWA server...")
			return app.Shutdown()
		},
	})
}
