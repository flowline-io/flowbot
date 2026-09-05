// Package automate provides the Automate JSON APIs for functions, pipeline, and workflow.
package automate

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/module"
	pkgpipeline "github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	pkgworkflow "github.com/flowline-io/flowbot/pkg/workflow"
)

// Name is the module identity and modules.automate config key.
const Name = "automate"

const (
	functionsGroup = "automate/functions"
	pipelineGroup  = "automate/pipeline"
	workflowGroup  = "automate/workflow"
)

var handler moduleHandler
var config configType

// Register registers the automate module handler.
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

// Init initializes the automate module. Enabled defaults to true when omitted.
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

// Webservice registers HTTP routes under /service/automate/{functions|pipeline|workflow}/*.
// Routes are mounted during handleRoutes, which runs before module Init, so
// registration must not depend on handler.initialized (same pattern as web/hub).
func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, functionsGroup, functionsRules)
	module.Webservice(app, pipelineGroup, pipelineRules)
	module.Webservice(app, workflowGroup, workflowRules)
}

// InitForE2E initializes the automate module handler for e2e testing.
// Subsequent calls are no-ops so Ginkgo BeforeEach can run once per spec.
func InitForE2E(configData json.RawMessage) error {
	if handler.initialized {
		return nil
	}
	return handler.Init(configData)
}

// MountForE2E mounts automate module routes onto the given Fiber app.
func MountForE2E(app *fiber.App) {
	handler.Webservice(app)
}

// Rules returns module rule sets.
func (moduleHandler) Rules() []any {
	return []any{functionsRules, pipelineRules, workflowRules}
}

// Input handles chat input (unused).
func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return nil, nil
}

func activeFunctionService() (*pkgfunctions.Service, error) {
	return requireActive(pkgfunctions.ActiveService(), "function service not ready")
}

func activePipelineService() (*pkgpipeline.Service, error) {
	return requireActive(pkgpipeline.ActiveService(), "pipeline service not ready")
}

func activeWorkflowService() (*pkgworkflow.Service, error) {
	return requireActive(pkgworkflow.ActiveService(), "workflow service not ready")
}

func requireActive[T any](svc *T, unavailableMsg string) (*T, error) {
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "%s", unavailableMsg)
	}
	return svc, nil
}

func requestUID(ctx fiber.Ctx) string {
	rc := route.GetRequestContext(ctx)
	if rc == nil || rc.UID.IsZero() {
		return ""
	}
	return rc.UID.String()
}
