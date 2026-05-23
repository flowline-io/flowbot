package example

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	abilityexample "github.com/flowline-io/flowbot/pkg/ability/example"
	adapter "github.com/flowline-io/flowbot/pkg/ability/example/example"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

const Name = "example"

var handler moduleHandler
var config configType

func Register() {
	module.Register(Name, &handler)
}

type moduleHandler struct {
	initialized bool
	module.Base
}

type configType struct {
	Enabled     bool   `json:"enabled"`
	Environment string `json:"environment"`
}

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
	// Register the example capability with the adapter.
	svc := adapter.New()
	if err := abilityexample.RegisterService("example", config.Environment, svc); err != nil {
		return fmt.Errorf("register example ability: %w", err)
	}
	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Bootstrap() error {
	return nil
}

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		formRules,
		pageRules,
		webserviceRules,
		webhookRules,
	}
}

func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}

func (moduleHandler) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(commandRules, ctx, content)
}

func (moduleHandler) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return module.RunForm(formRules, ctx, values)
}

func (moduleHandler) Page(ctx types.Context, flag string, args types.KV) (string, error) {
	return module.RunPage(pageRules, ctx, flag, args)
}
