package workflow

import (
	"encoding/json"
	"errors"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

const Name = "workflow"

var handler bot

func init() {
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
	if err := jsoniter.Unmarshal(jsonconf, &config); err != nil {
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

func (b bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		formRules,
	}
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (bot) Webservice(app *fiber.App) {
	bots.Webservice(app, Name, webserviceRules)
}

func (b bot) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return bots.RunForm(formRules, ctx, values)
}
