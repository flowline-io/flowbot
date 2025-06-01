package server

import (
	"context"
	"fmt"
	"github.com/VictoriaMetrics/metrics"
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/rules"
	"github.com/flowline-io/flowbot/internal/rules/components"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
	"github.com/gofiber/fiber/v3"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/endpoint"
	"time"
)

var (
	// swagger
	swagHandler fiber.Handler
)

func initializeLog() error {
	flog.Init(false)
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
	return metrics.InitPushWithOptions(
		context.Background(),
		fmt.Sprintf("%s/api/v1/import/prometheus", config.App.Metrics.Endpoint),
		10*time.Second,
		true,
		&metrics.PushOptions{
			ExtraLabels: fmt.Sprintf(`instance="flowbot",version="%s"`, version.Buildtags),
		},
	)
}

func initializeRuleEngine(app *fiber.App) error {
	// register functions
	rules.RegisterFunctions()

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
