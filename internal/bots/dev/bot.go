package dev

import (
	"encoding/json"
	"errors"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/ruleset/instruct"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	"net/http"
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
	if err := json.Unmarshal(jsonconf, &config); err != nil {
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

func (bot) OnEvent() error {
	// todo ExampleEvent
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
		conditionRules,
		actionRules,
		instructRules,
		sessionRules,
		pageRules,
		agentRules,
		webserviceRules,
		workflowRules,
		webhookRules,
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

func (b bot) Action(ctx types.Context, option string) (types.MsgPayload, error) {
	return bots.RunAction(actionRules, ctx, option)
}

func (b bot) Cron(send types.SendFunc) (*cron.Ruleset, error) {
	return bots.RunCron(cronRules, Name, send)
}

func (b bot) Condition(ctx types.Context, forwarded types.MsgPayload) (types.MsgPayload, error) {
	return bots.RunCondition(conditionRules, ctx, forwarded)
}

func (b bot) Group(ctx types.Context, head types.KV, content interface{}) (types.MsgPayload, error) {
	return bots.RunGroup(eventRules, ctx, head, content)
}

func (b bot) Agent(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return bots.RunAgent(AgentVersion, agentRules, ctx, content)
}

func (b bot) Session(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunSession(sessionRules, ctx, content)
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

func (b bot) Webhook(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return bots.RunWebhook(webhookRules, ctx, content)
}
