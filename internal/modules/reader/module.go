package reader

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

const Name = "reader"

var handler moduleHandler

func Register() {
	module.Register(Name, &handler)
}

type moduleHandler struct {
	initialized bool
	module.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	// Check if the handler is already initialized
	if handler.initialized {
		return errors.New("already initialized")
	}

	var config configType
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
	return nil
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		cronRules,
		webhookRules,
		webserviceRules,
	}
}

func (moduleHandler) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(commandRules, ctx, content)
}

func (moduleHandler) Cron() (*cron.Ruleset, error) {
	return module.RunCron(cronRules, Name)
}

func (moduleHandler) Webhook(ctx types.Context, data []byte) (types.MsgPayload, error) {
	return module.RunWebhook(webhookRules, ctx, data)
}
