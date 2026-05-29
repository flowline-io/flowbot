// Package web provides a web UI module with server-rendered HTML pages.
package web

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
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
	module.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
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
	handler.initialized = true
	return nil
}

// IsReady checks if the web module is initialized.
func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Bootstrap performs post-initialization setup.
func (moduleHandler) Bootstrap() error {
	return nil
}

// Webservice mounts web module routes on the fiber app.
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("./public"))
	module.Webservice(app, Name, webserviceRules)
}

// Rules returns the web module rule definitions.
func (moduleHandler) Rules() []any {
	return []any{webserviceRules}
}
