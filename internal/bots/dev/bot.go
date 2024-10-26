package dev

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/ruleset/instruct"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

const Name = "dev"

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

func (bot) Bootstrap() error {
	// load setting rule
	formRules = append(formRules, bots.SettingCovertForm(Name, settingRules))

	return nil
}

func (bot) Webapp() func(rw http.ResponseWriter, req *http.Request) {
	return webapp
}

func (bot) Webservice(app *fiber.App) {
	bots.Webservice(app, Name, webserviceRules)
}

func (b bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		formRules,
		instructRules,
		pageRules,
		agentRules,
		webserviceRules,
		workflowRules,
		webhookRules,
		langchainRules,
	}
}

func (b bot) Input(_ types.Context, _ types.KV, _ interface{}) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return bots.RunForm(formRules, ctx, values)
}

func (b bot) Cron() (*cron.Ruleset, error) {
	return bots.RunCron(cronRules, Name)
}

func (b bot) Agent(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return bots.RunAgent(AgentVersion, agentRules, ctx, content)
}

func (b bot) Instruct() (instruct.Ruleset, error) {
	return instructRules, nil
}

func (b bot) Page(ctx types.Context, flag string) (string, error) {
	return bots.RunPage(pageRules, ctx, flag)
}

func (b bot) Workflow(ctx types.Context, input types.KV) (types.KV, error) {
	return bots.RunWorkflow(workflowRules, ctx, input)
}

func (b bot) Webhook(ctx types.Context, method string, data []byte) (types.MsgPayload, error) {
	return bots.RunWebhook(webhookRules, ctx, method, data)
}

func (b bot) LangChain(ctx types.Context, args types.KV) (string, error) {
	return bots.RunLangChain(langchainRules, ctx, args)
}
