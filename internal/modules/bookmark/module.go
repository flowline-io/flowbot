package bookmark

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/gofiber/fiber/v3"
)

const Name = "bookmark"

var handler moduleHandler
var Config configType

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

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	// Check if the handler is already initialized
	if handler.initialized {
		return errors.New("already initialized")
	}

	if err := sonic.Unmarshal(jsonconf, &Config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !Config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		cronRules,
		eventRules,
		webserviceRules,
	}
}

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

func (moduleHandler) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(commandRules, ctx, content)
}

func (moduleHandler) Cron() (*cron.Ruleset, error) {
	return module.RunCron(cronRules, Name)
}

func (moduleHandler) Event(ctx types.Context, param types.KV) error {
	return module.RunEvent(eventRules, ctx, param)
}
