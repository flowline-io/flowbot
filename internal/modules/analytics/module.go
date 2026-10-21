// Package analytics is Flowbot's personal and ops statistics module.
package analytics

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
)

// Name is the module registry key.
const Name = "analytics"

var handler moduleHandler
var config configType
var analyticsService *Service
var serviceListeners []func(*Service)

type configType struct {
	Enabled bool `json:"enabled"`
}

type moduleHandler struct {
	initialized bool
	module.Base
}

// Register registers the analytics module.
func Register() {
	module.Register(Name, &handler)
}

// OnService registers a listener invoked whenever the analytics service instance is assigned.
func OnService(fn func(*Service)) {
	if fn == nil {
		return
	}
	serviceListeners = append(serviceListeners, fn)
	if analyticsService != nil {
		fn(analyticsService)
	}
}

func setActiveService(s *Service) {
	analyticsService = s
	for _, fn := range serviceListeners {
		fn(s)
	}
}

// ActiveService returns the current analytics service, or nil when disabled.
func ActiveService() *Service {
	return analyticsService
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if len(jsonconf) > 0 {
		if err := sonic.Unmarshal(jsonconf, &config); err != nil {
			return fmt.Errorf("failed to parse analytics config: %w", err)
		}
	} else {
		config.Enabled = true
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	setActiveService(NewService())
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
	setActiveService(NewService())
	flog.Info("analytics: bootstrap complete")
	return nil
}

func (moduleHandler) Webservice(_ *fiber.App) {}

func (moduleHandler) Rules() []any { return nil }
