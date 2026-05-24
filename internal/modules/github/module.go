package github

import (
	"encoding/json"
	"errors"
	"fmt"

	abilitygithub "github.com/flowline-io/flowbot/pkg/ability/github"
	"github.com/bytedance/sonic"
	githubadapter "github.com/flowline-io/flowbot/pkg/ability/github/github"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

const Name = "github"

var handler moduleHandler
var Config configType

func Register() {
	module.Register(Name, &handler)
}

type moduleHandler struct {
	initialized bool
	module.Base
}

type configType struct {
	Enabled bool   `json:"enabled"`
	Backend string `json:"backend"`
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}

	if err := sonic.Unmarshal(jsonconf, &Config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if !Config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}

	// Register the GitHub capability with the adapter.
	backend := Config.Backend
	if backend == "" {
		backend = "github"
	}
	svc := githubadapter.New()
	if err := abilitygithub.RegisterService(backend, "", svc); err != nil {
		return fmt.Errorf("register github ability: %w", err)
	}

	handler.initialized = true

	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		formRules,
	}
}

func (moduleHandler) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(commandRules, ctx, content)
}

func (moduleHandler) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return module.RunForm(formRules, ctx, values)
}
