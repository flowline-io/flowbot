package server

import (
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/gofiber/fiber/v3"
)

var (
	// swagger
	swagHandler fiber.Handler
)

func initializeLog() error {
	cfg := config.App.Log
	fc := flog.Config{
		Level:        cfg.Level,
		Caller:       cfg.Caller,
		StackTrace:   cfg.StackTrace,
		JSONOutput:   cfg.JSONOutput,
		FileLog:      cfg.FileLog,
		FileLogPath:  cfg.FileLogPath,
		AlarmEnabled: config.App.Alarm.Enabled,
		ModuleLevel:  cfg.ModuleLevel,
	}
	if cfg.Sampling != nil {
		fc.Sampling = &flog.SamplingConfig{
			Burst:  cfg.Sampling.Burst,
			Period: time.Duration(cfg.Sampling.Period) * time.Second,
		}
	}
	if cfg.Rotation != nil {
		fc.Rotation = &flog.RotationConfig{
			MaxSize:    cfg.Rotation.MaxSize,
			MaxAge:     cfg.Rotation.MaxAge,
			MaxBackups: cfg.Rotation.MaxBackups,
			Compress:   cfg.Rotation.Compress,
		}
	}
	flog.Init(fc)
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
