package search

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/gofiber/fiber/v3"
)

const Name = "search"

var handler bot

func Register() {
	chatbot.Register(Name, &handler)
}

type bot struct {
	initialized bool
	chatbot.Base
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

func (bot) Bootstrap() error {
	return nil
}

func (bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		cronRules,
		collectRules,
		webserviceRules,
		pageRules,
	}
}

func (bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return chatbot.RunCommand(commandRules, ctx, content)
}

func (bot) Cron() (*cron.Ruleset, error) {
	return chatbot.RunCron(cronRules, Name)
}

func (bot) Collect(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return chatbot.RunCollect(collectRules, ctx, content)
}

func (bot) Webservice(app *fiber.App) {
	chatbot.Webservice(app, Name, webserviceRules)
}

func (bot) Page(ctx types.Context, flag string, args types.KV) (string, error) {
	return chatbot.RunPage(pageRules, ctx, flag, args)
}
