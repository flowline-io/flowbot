// Package hub implements the hub management module providing chat commands
// for health checks, app management, and resource tag query endpoints.
package hub

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

const Name = "hub"

var handler moduleHandler
var rcStore *store.ResourceChainStore

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
	if handler.initialized {
		return errors.New("already initialized")
	}

	var config configType
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}

	if store.Database == nil {
		return errors.New("store database not available")
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok || client == nil {
		return errors.New("store client not available")
	}
	rcStore = store.NewResourceChainStore(client)

	handler.initialized = true

	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Bootstrap() error { return nil }

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		webserviceRules,
	}
}

func (moduleHandler) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(commandRules, ctx, content)
}

func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}
