package github

import (
	"encoding/json"
	"errors"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/flog"
	jsoniter "github.com/json-iterator/go"
)

const Name = "github"

var handler bot
var Config configType

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

	if err := jsoniter.Unmarshal(jsonconf, &Config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !Config.Enabled {
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

func (b bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		formRules,
	}
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Cron() (*cron.Ruleset, error) {
	return bots.RunCron(cronRules, Name)
}

func (b bot) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return bots.RunForm(formRules, ctx, values)
}
