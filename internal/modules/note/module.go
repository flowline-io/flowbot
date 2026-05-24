// Package note implements the note module for interacting with note-taking systems.
package note

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
)

// Name is the module identifier used in configuration and registration.
const Name = "note"

var handler moduleHandler
var config configType

// Register registers the note module with the module system.
func Register() {
	module.Register(Name, &handler)
}

type moduleHandler struct {
	module.Base
	initialized bool
}

type configType struct {
	Enabled bool `json:"enabled"`
}

// Init initializes the note module from its JSON configuration.
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

// IsReady reports whether the module is initialized and enabled.
func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Webservice registers the note HTTP routes on the Fiber app.
func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

// Rules returns the module's interaction rule definitions.
func (moduleHandler) Rules() []any {
	return []any{
		webserviceRules,
	}
}
