package server

import (
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/rules"
	"github.com/flowline-io/flowbot/internal/rules/components"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/gofiber/fiber/v3"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/endpoint"
)

var (
	// swagger
	swagHandler fiber.Handler
)

func initializeLog() error {
	flog.Init(false, config.App.Alarm.Enabled)
	flog.SetLevel(config.App.Log.Level)
	return nil
}

func initializeTimezone() error {
	_, err := time.LoadLocation("Local")
	if err != nil {
		return fmt.Errorf("load time location error, %w", err)
	}
	return nil
}

func initializeMedia() error {
	// Media
	if config.App.Media != nil {
		if config.App.Media.UseHandler == "" {
			config.App.Media = nil
		} else {
			globals.maxFileUploadSize = config.App.Media.MaxFileUploadSize
			if config.App.Media.Handlers != nil {
				var conf string
				if params := config.App.Media.Handlers[config.App.Media.UseHandler]; params != nil {
					data, err := sonic.Marshal(params)
					if err != nil {
						return fmt.Errorf("failed to marshal media handler, %w", err)
					}
					conf = string(data)
				}
				if err := store.UseMediaHandler(config.App.Media.UseHandler, conf); err != nil {
					return fmt.Errorf("failed to init media handler, %w", err)
				}
			}
		}
	}
	return nil
}

func initializeMetrics() error {
	if !config.App.Metrics.Enabled {
		flog.Info("metrics disabled")
		return nil
	}

	return stats.Init(&stats.MetricsConfig{
		PushGatewayURL: config.App.Metrics.Endpoint,
		PushInterval:   time.Duration(15) * time.Second,
	})
}

func initializeRuleEngine(app *fiber.App) error {
	// register components
	err := rulego.Registry.Register(&components.CommandNode{})
	if err != nil {
		return err
	}
	err = rulego.Registry.Register(&components.DataNode{})
	if err != nil {
		return err
	}
	err = rulego.Registry.Register(&components.FunctionsNode{})
	if err != nil {
		return err
	}
	err = rulego.Registry.Register(&components.DefaultUserNode{})
	if err != nil {
		return err
	}

	// register functions
	rules.RegisterFunctions()

	// register endpoints
	err = endpoint.Registry.Register(&RestEndpoint{})

	err = rules.InitEngine()
	if err != nil {
		return err
	}
	err = rules.InitEndpoint()
	if err != nil {
		return err
	}

	return nil
}
