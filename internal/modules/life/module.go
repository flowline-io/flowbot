// Package life is Flowbot's solo gamified productivity module.
package life

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
)

// Name is the module registry key.
const Name = "life"

var handler moduleHandler
var config configType
var lifeService *Service
var serviceListeners []func(*Service)

type configType struct {
	Enabled      bool   `json:"enabled"`
	DefaultClass string `json:"default_class"`
}

type moduleHandler struct {
	initialized bool
	module.Base
	stopLore chan struct{}
}

// Register registers the life module.
func Register() {
	module.Register(Name, &handler)
}

// OnService registers a listener invoked whenever the Life service instance is assigned.
func OnService(fn func(*Service)) {
	if fn == nil {
		return
	}
	serviceListeners = append(serviceListeners, fn)
	if lifeService != nil {
		fn(lifeService)
	}
}

func setActiveService(s *Service) {
	lifeService = s
	for _, fn := range serviceListeners {
		fn(s)
	}
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if len(jsonconf) > 0 {
		if err := sonic.Unmarshal(jsonconf, &config); err != nil {
			return fmt.Errorf("failed to parse life config: %w", err)
		}
	} else {
		config.Enabled = true
	}
	if config.DefaultClass == "" {
		config.DefaultClass = "Architect"
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	if err := lifecap.Register(lifecap.NewLLM()); err != nil {
		return fmt.Errorf("register life capability: %w", err)
	}
	setActiveService(NewService(store.LifeStoreFromDB()))
	handler.initialized = true
	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Bootstrap() error {
	if !handler.initialized {
		return nil
	}
	ctx := context.Background()
	ls := store.LifeStoreFromDB()
	if ls.Client() == nil {
		flog.Warn("life: bootstrap skipped, store unavailable")
		return nil
	}
	lifeService = NewService(ls)
	if err := seedCatalog(ctx, ls); err != nil {
		return fmt.Errorf("life: seed catalog: %w", err)
	}
	setActiveService(lifeService)
	flog.Info("life: bootstrap complete (catalog seeded, lore worker started)")
	if handler.stopLore != nil {
		close(handler.stopLore)
	}
	handler.stopLore = make(chan struct{})
	go loreLoop(handler.stopLore)
	return nil
}

func (moduleHandler) Webservice(_ *fiber.App) {}

func (moduleHandler) Rules() []any { return nil }

func loreLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if lifeService == nil {
				continue
			}
			if _, err := lifeService.ProcessPendingLore(context.Background()); err != nil {
				flog.Warn("life: process lore: %v", err)
			}
		}
	}
}

// ActiveService returns the module service for web injection.
func ActiveService() *Service {
	if lifeService != nil {
		return lifeService
	}
	return NewService(store.LifeStoreFromDB())
}
