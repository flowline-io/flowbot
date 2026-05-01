package dev

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/instruct"
	"github.com/gofiber/fiber/v3"
)

const Name = "dev"

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
	// Check if the handler is already initialized
	if handler.initialized {
		return errors.New("already initialized")
	}

	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Bootstrap() error {
	// load setting rule
	formRules = append(formRules, module.SettingCovertForm(Name, settingRules))

	return nil
}

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		formRules,
		instructRules,
		pageRules,
		collectRules,
		webserviceRules,
		webhookRules,
		eventRules,
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

func (moduleHandler) Cron() (*cron.Ruleset, error) {
	return module.RunCron(cronRules, Name)
}

func (moduleHandler) Collect(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return module.RunCollect(collectRules, ctx, content)
}

func (moduleHandler) Instruct() (instruct.Ruleset, error) {
	return instructRules, nil
}

func (moduleHandler) Page(ctx types.Context, flag string, args types.KV) (string, error) {
	return module.RunPage(pageRules, ctx, flag, args)
}

func (moduleHandler) Webhook(ctx types.Context, data []byte) (types.MsgPayload, error) {
	return module.RunWebhook(webhookRules, ctx, data)
}

func (moduleHandler) Event(ctx types.Context, param types.KV) error {
	return module.RunEvent(eventRules, ctx, param)
}
