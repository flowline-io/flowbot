package bookmark

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

const Name = "bookmark"

var handler bot
var Config configType

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

	if err := sonic.Unmarshal(jsonconf, &Config); err != nil {
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

func (b bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		cronRules,
		eventRules,
	}
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Cron() (*cron.Ruleset, error) {
	return bots.RunCron(cronRules, Name)
}

func (b bot) Event(ctx types.Context, param types.KV) error {
	return bots.RunEvent(eventRules, ctx, param)
}
