// Package web provides a web UI module with server-rendered HTML pages.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	webassets "github.com/flowline-io/flowbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/flowline-io/flowbot/internal/store"
)

const Name = "web"

var handler moduleHandler
var config configType

// Register registers the web module handler.
func Register() {
	module.Register(Name, &handler)
}

type moduleHandler struct {
	initialized bool
	authConfig  AuthConfig
	module.Base
}

type configType struct {
	Enabled bool       `json:"enabled"`
	Auth    AuthConfig `json:"auth"`
}

// Init initializes the web module with the given JSON configuration.
func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	if err := validateAuthConfig(config.Auth); err != nil {
		return err
	}
	handler.initialized = true
	handler.authConfig = config.Auth
	return nil
}

// IsReady checks if the web module is initialized.
func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Bootstrap performs post-initialization setup.
func (moduleHandler) Bootstrap() error {
	if !handler.initialized {
		return nil
	}
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			es := store.NewEventStore(client)
			sources, err := es.ListDistinctEventSources(context.Background(), 30*24*time.Hour)
			if err == nil {
				distinctTypes, err2 := es.ListDistinctEventTypes(context.Background(), 30*24*time.Hour)
				if err2 == nil {
					types.EventFilterCache.Hydrate(sources, distinctTypes)
				}
			}
		}
	}
	return nil
}

// Webservice mounts web module routes on the fiber app.
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	for _, rules := range allWebserviceRules {
		module.Webservice(app, Name, rules)
	}
}

// Rules returns the web module rule definitions as one entry per route group in allWebserviceRules.
func (moduleHandler) Rules() []any {
	out := make([]any, len(allWebserviceRules))
	for i, rules := range allWebserviceRules {
		out[i] = rules
	}
	return out
}

// InitForE2E initializes the web module handler for e2e testing.
// It calls the package-level handler.Init with the provided JSON config,
// bypassing the uber/fx dependency injection used in production.
func InitForE2E(configData json.RawMessage) error {
	return handler.Init(configData)
}

// MountForE2E mounts web module routes onto the given Fiber app.
func MountForE2E(app *fiber.App) {
	handler.Webservice(app)
}
