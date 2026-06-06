package flog

import (
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/config"
)

// FxModule provides the zerolog.Logger to the Fx dependency injection graph.
// It reads log configuration from *config.Type and calls Init before returning.
var FxModule = fx.Module("flog",
	fx.Provide(func(cfg *config.Type) (zerolog.Logger, error) {
		logCfg := cfg.Log
		fc := Config{
			Level:       logCfg.Level,
			Caller:      logCfg.Caller,
			StackTrace:  logCfg.StackTrace,
			JSONOutput:  logCfg.JSONOutput,
			FileLog:     logCfg.FileLog,
			FileLogPath: logCfg.FileLogPath,
			ModuleLevel: logCfg.ModuleLevel,
		}
		if logCfg.Sampling != nil {
			fc.Sampling = &SamplingConfig{
				Burst:  logCfg.Sampling.Burst,
				Period: time.Duration(logCfg.Sampling.Period) * time.Second,
			}
		}
		if logCfg.Rotation != nil {
			fc.Rotation = &RotationConfig{
				MaxSize:    logCfg.Rotation.MaxSize,
				MaxAge:     logCfg.Rotation.MaxAge,
				MaxBackups: logCfg.Rotation.MaxBackups,
				Compress:   logCfg.Rotation.Compress,
			}
		}
		Init(fc)
		return GetLogger(), nil
	}),
)
