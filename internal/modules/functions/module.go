// Package functions provides the named-functions HTTP API module.
package functions

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Name is the module identity and /service/{name} route group.
const Name = "functions"

var handler moduleHandler
var config configType

// Register registers the functions module handler.
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

// Init initializes the functions module. Enabled defaults to true when omitted.
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

// Webservice registers HTTP routes under /service/functions/*.
func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

// InitForE2E initializes the functions module handler for e2e testing.
// Subsequent calls are no-ops so Ginkgo BeforeEach can run once per spec.
func InitForE2E(configData json.RawMessage) error {
	if handler.initialized {
		return nil
	}
	return handler.Init(configData)
}

// MountForE2E mounts functions module routes onto the given Fiber app.
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

func activeService() (*pkgfunctions.Service, error) {
	svc := pkgfunctions.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	return svc, nil
}
