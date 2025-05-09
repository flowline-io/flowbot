package dev

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/instruct"
	"github.com/gofiber/fiber/v2"
)

const Name = "dev"

var handler bot
var config configType

func Register() {
	chatbot.Register(Name, &handler)
}

type bot struct {
	initialized bool
	chatbot.Base
}

type configType struct {
	Enabled     bool   `json:"enabled"`
	Environment string `json:"environment"`
}

func (bot) Init(jsonconf json.RawMessage) error {
	// Check if the handler is already initialized
	if handler.initialized {
		return errors.New("already initialized")
	}

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

func (bot) Bootstrap() error {
	// load setting rule
	formRules = append(formRules, chatbot.SettingCovertForm(Name, settingRules))

	return nil
}

func (bot) Webservice(app *fiber.App) {
	chatbot.Webservice(app, Name, webserviceRules)
}

func (bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		formRules,
		instructRules,
		pageRules,
		collectRules,
		webserviceRules,
		workflowRules,
		webhookRules,
		toolRules,
		eventRules,
	}
}

func (bot) Input(_ types.Context, _ types.KV, _ interface{}) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}

func (bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return chatbot.RunCommand(commandRules, ctx, content)
}

func (bot) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return chatbot.RunForm(formRules, ctx, values)
}

func (bot) Cron() (*cron.Ruleset, error) {
	return chatbot.RunCron(cronRules, Name)
}

func (bot) Collect(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return chatbot.RunCollect(collectRules, ctx, content)
}

func (bot) Instruct() (instruct.Ruleset, error) {
	return instructRules, nil
}

func (bot) Page(ctx types.Context, flag string, args types.KV) (string, error) {
	return chatbot.RunPage(pageRules, ctx, flag, args)
}

func (bot) Workflow(ctx types.Context, input types.KV) (types.KV, error) {
	return chatbot.RunWorkflow(workflowRules, ctx, input)
}

func (bot) Webhook(ctx types.Context, data []byte) (types.MsgPayload, error) {
	return chatbot.RunWebhook(webhookRules, ctx, data)
}

func (bot) Tool(ctx types.Context, argumentsInJSON string) (string, error) {
	return chatbot.RunTool(toolRules, ctx, argumentsInJSON)
}

func (bot) Event(ctx types.Context, param types.KV) error {
	return chatbot.RunEvent(eventRules, ctx, param)
}
