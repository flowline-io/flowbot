package server

import (
	"encoding/json"
	"errors"
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/cron"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
)

const Name = "server"

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
		logs.Info.Printf("bot %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (bot) IsReady() bool {
	return handler.initialized
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Agent(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return bots.RunAgent(AgentVersion, agentRules, ctx, content)
}

func (b bot) Cron(send types.SendFunc) (*cron.Ruleset, error) {
	return bots.RunCron(cronRules, Name, send)
}
