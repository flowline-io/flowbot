package profiling

import (
	"context"
	"fmt"

	"github.com/grafana/pyroscope-go"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// pyroscopeLogger adapts flog to the pyroscope.Logger interface.
type pyroscopeLogger struct{}

func (pyroscopeLogger) Infof(format string, a ...any)  { flog.Info(format, a...) }
func (pyroscopeLogger) Debugf(format string, a ...any) { flog.Debug(format, a...) }
func (pyroscopeLogger) Errorf(format string, a ...any) {
	flog.Err(fmt.Errorf(format, a...))
}

// NewProfiler starts Pyroscope continuous profiling and registers shutdown via fx lifecycle.
func NewProfiler(lc fx.Lifecycle) error {
	cfg := config.App.Profiling
	if !cfg.Enabled {
		flog.Info("profiling disabled, skipping Pyroscope init")
		return nil
	}

	pyroscopeCfg := pyroscope.Config{
		ApplicationName: cfg.ServiceName,
		ServerAddress:   cfg.ServerAddress,
		Logger:          pyroscopeLogger{},
		Tags: map[string]string{
			"service.name": cfg.ServiceName,
			"environment":  cfg.Environment,
		},
		ProfileTypes: parseProfileTypes(cfg.ProfileTypes),
	}

	if pyroscopeCfg.ApplicationName == "" {
		pyroscopeCfg.ApplicationName = "flowbot"
	}
	if pyroscopeCfg.ServerAddress == "" {
		pyroscopeCfg.ServerAddress = "http://localhost:4040"
	}
	if pyroscopeCfg.Tags["environment"] == "" {
		pyroscopeCfg.Tags["environment"] = "development"
	}
	if len(pyroscopeCfg.ProfileTypes) == 0 {
		pyroscopeCfg.ProfileTypes = pyroscope.DefaultProfileTypes
	}

	pyroscopeCfg.Tags["service.name"] = pyroscopeCfg.ApplicationName

	var profiler *pyroscope.Profiler

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p, err := pyroscope.Start(pyroscopeCfg)
			if err != nil {
				return fmt.Errorf("failed to start pyroscope profiler: %w", err)
			}
			profiler = p
			flog.Info("pyroscope profiler started: addr=%s service=%s env=%s types=%v",
				pyroscopeCfg.ServerAddress, pyroscopeCfg.ApplicationName,
				pyroscopeCfg.Tags["environment"], profileTypeNames(pyroscopeCfg.ProfileTypes))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if profiler != nil {
				if err := profiler.Stop(); err != nil {
					flog.Err(fmt.Errorf("pyroscope profiler stop error: %w", err))
				} else {
					flog.Info("pyroscope profiler stopped")
				}
			}
			return nil
		},
	})

	return nil
}

// parseProfileTypes converts config string names to pyroscope.ProfileType values.
func parseProfileTypes(names []string) []pyroscope.ProfileType {
	if len(names) == 0 {
		return nil
	}
	types := make([]pyroscope.ProfileType, 0, len(names))
	for _, n := range names {
		types = append(types, pyroscope.ProfileType(n))
	}
	return types
}

// profileTypeNames returns the string names for logging.
func profileTypeNames(types []pyroscope.ProfileType) []string {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	return names
}
