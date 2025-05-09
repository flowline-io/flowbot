package user

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v2"
)

const Name = "user"

var handler bot

func Register() {
	bots.Register(Name, &handler)
}

type bot struct {
	initialized bool
	bots.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func (bot) Init(jsonconf json.RawMessage) error {
	// Check if the handler is already initialized
	if handler.initialized {
		return errors.New("already initialized")
	}

	var config configType
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !config.Enabled {
		flog.Info("bot %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (bot) IsReady() bool {
	return handler.initialized
}

func (bot) Webservice(app *fiber.App) {
	bots.Webservice(app, Name, webserviceRules)
}

func (b bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		webserviceRules,
	}
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}
