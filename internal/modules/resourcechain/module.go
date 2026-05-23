// Package resourcechain provides resource tag query and relation endpoints.
package resourcechain

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

const Name = "resourcechain"

var handler moduleHandler
var config configType
var rcStore *store.ResourceChainStore

type moduleHandler struct {
	initialized bool
	module.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func Register() {
	module.Register(Name, &handler)
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
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
	return []any{webserviceRules}
}

func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}
