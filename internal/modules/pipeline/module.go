// Package pipeline provides the platform pipeline HTTP API module.
package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	pkgpipeline "github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

const Name = "pipeline"

var handler moduleHandler
var config configType

// Register registers the pipeline module handler.
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

// Init initializes the pipeline module. Enabled defaults to true when omitted.
func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	config.Enabled = true
	if len(jsonconf) > 0 && string(jsonconf) != "null" {
		var raw map[string]any
		if err := sonic.Unmarshal(jsonconf, &raw); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
		if err := sonic.Unmarshal(jsonconf, &config); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
		if _, ok := raw["enabled"]; !ok {
			config.Enabled = true
		}
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	handler.initialized = true
	return nil
}

// IsReady reports whether the module is initialized.
func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Bootstrap performs post-initialization setup.
func (moduleHandler) Bootstrap() error {
	return nil
}

// Webservice registers HTTP routes under /service/pipeline/*.
// Routes are mounted during handleRoutes, which runs before module Init, so
// registration must not depend on handler.initialized (same pattern as web/hub).
func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

// InitForE2E initializes the pipeline module handler for e2e testing.
func InitForE2E(configData json.RawMessage) error {
	return handler.Init(configData)
}

// MountForE2E mounts pipeline module routes onto the given Fiber app.
func MountForE2E(app *fiber.App) {
	handler.Webservice(app)
}

// Rules returns module rule sets.
func (moduleHandler) Rules() []any {
	return []any{webserviceRules}
}

// Input handles chat input (unused).
func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return nil, nil
}

func activeService() (*pkgpipeline.Service, error) {
	svc := pkgpipeline.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	return svc, nil
}
