// Package lineage serves audit and lineage read APIs.
package lineage

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/module"
)

// Name is the module id registered with the process and used as the route group.
const Name = "lineage"

type moduleHandler struct {
	module.Base
}

// Register adds the lineage module.
func Register() {
	module.Register(Name, moduleHandler{})
}

func (moduleHandler) Init(json.RawMessage) error {
	return nil
}

func (moduleHandler) IsReady() bool {
	return true
}

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, "audit", auditRules)
}

// MountForE2E mounts lineage routes onto app.
func MountForE2E(app *fiber.App) {
	moduleHandler{}.Webservice(app)
}
