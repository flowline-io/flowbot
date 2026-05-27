// Package hub implements the hub management module providing chat commands
// for health checks, app management, resource tag query endpoints, and
// consolidated bookmark, github, kanban, memo, note, and reader capabilities.
package hub

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/ability"
	karakeepAdapter "github.com/flowline-io/flowbot/pkg/ability/bookmark/karakeep"
	exampleAdapter "github.com/flowline-io/flowbot/pkg/ability/example/example"
	abilityforge "github.com/flowline-io/flowbot/pkg/ability/forge"
	giteaAdapter "github.com/flowline-io/flowbot/pkg/ability/forge/gitea"
	abilitygithub "github.com/flowline-io/flowbot/pkg/ability/github"
	githubadapter "github.com/flowline-io/flowbot/pkg/ability/github/github"
	kanboardAdapter "github.com/flowline-io/flowbot/pkg/ability/kanban/kanboard"
	abilitymemo "github.com/flowline-io/flowbot/pkg/ability/memo"
	memosAdapter "github.com/flowline-io/flowbot/pkg/ability/memo/memos"
	triliumAdapter "github.com/flowline-io/flowbot/pkg/ability/note/trilium"
	minifluxAdapter "github.com/flowline-io/flowbot/pkg/ability/reader/miniflux"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
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

	// Hub resource chain store
	if store.Database == nil {
		return errors.New("store database not available")
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok || client == nil {
		return errors.New("store client not available")
	}
	rcStore = store.NewResourceChainStore(client)

	// Register the GitHub capability with the adapter
	backend := githubConfig.Backend
	if backend == "" {
		backend = "github"
	}
	svc := githubadapter.New()
	if err := abilitygithub.RegisterService(backend, "", svc); err != nil {
		return fmt.Errorf("register github ability: %w", err)
	}

	// Register the forge capability with the Gitea adapter
	forgeBackend := "gitea"
	forgeSvc := giteaAdapter.New()
	if err := abilityforge.RegisterService(forgeBackend, "", forgeSvc); err != nil {
		return fmt.Errorf("register forge ability: %w", err)
	}

	// Register the memo capability with the Memos adapter
	memoBackend := "memos"
	memoSvc := memosAdapter.New()
	if memoSvc != nil {
		if err := abilitymemo.RegisterService(memoBackend, "", memoSvc); err != nil {
			return fmt.Errorf("register memo ability: %w", err)
		}
	}

	handler.initialized = true

	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Bootstrap registers the Karakeep webhook converter with the EventSourceManager.
func (moduleHandler) Bootstrap() error {
	if !handler.initialized {
		return nil
	}
	mgr := ability.GetEventSourceManager()
	if mgr == nil {
		return fmt.Errorf("hub: event source manager not initialized")
	}
	mgr.RegisterWebhook(karakeepAdapter.NewWebhook())
	flog.Info("hub: registered karakeep webhook on /webhook/provider/karakeep/events")
	mgr.RegisterWebhook(minifluxAdapter.NewWebhook())
	flog.Info("hub: registered miniflux webhook on /webhook/provider/miniflux/events")
	mgr.RegisterWebhook(giteaAdapter.NewGiteaWebhook())
	flog.Info("hub: registered gitea webhook on /webhook/provider/gitea/events")
	mgr.RegisterWebhook(memosAdapter.NewWebhook())
	flog.Info("hub: registered memos webhook on /webhook/provider/memos/events")
	mgr.RegisterWebhook(kanboardAdapter.NewWebhook())
	flog.Info("hub: registered kanboard webhook on /webhook/provider/kanboard/events")

	// Pollers
	mgr.RegisterPolling(exampleAdapter.NewPoller())
	flog.Info("hub: registered example poller")
	mgr.RegisterPolling(triliumAdapter.NewPoller())
	flog.Info("hub: registered trilium note poller")
	return nil
}

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, hubWebserviceRules)
	module.Webservice(app, "bookmark", bookmarkWebserviceRules)
	module.Webservice(app, "kanban", kanbanWebserviceRules)
	module.Webservice(app, "note", noteWebserviceRules)
	module.Webservice(app, "reader", readerWebserviceRules)
	module.Webservice(app, "forge", forgeWebserviceRules)
	module.Webservice(app, "github", githubWebserviceRules)
	module.Webservice(app, "memo", memoWebserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		webserviceRules,
		formRules,
	}
}

func (moduleHandler) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(commandRules, ctx, content)
}

func (moduleHandler) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return module.RunForm(formRules, ctx, values)
}

func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}

// Form rules for github module (formerly separate).
var formRules []form.Rule
