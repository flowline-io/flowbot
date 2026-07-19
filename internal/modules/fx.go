// Package modules provides fx dependency injection registration for all modules.
package modules

import (
	"context"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules/example"
	"github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/plugin/manager"
)

// Modules registers all interaction modules.
var Modules = fx.Options(
	fx.Provide(func(log zerolog.Logger, cfg *config.Type) *manager.PluginManager {
		mgr := manager.NewPluginManager(cfg.Plugins, log)
		if cfg.Plugins != nil && cfg.Plugins.Enabled {
			if err := mgr.Init(context.Background(), cfg.Plugins.Config); err != nil {
				log.Error().Err(err).Msg("plugin manager init failed")
			}
		}
		return mgr
	}),

	fx.Invoke(func(mgr *manager.PluginManager, lc fx.Lifecycle, log zerolog.Logger) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				for _, inst := range mgr.List() {
					if err := mgr.UnloadPlugin(ctx, inst.Identity); err != nil {
						log.Error().Err(err).Str("plugin", inst.Identity).Msg("unload plugin failed")
					}
				}
				return nil
			},
		})
	}),

	fx.Invoke(
		example.Register,
		hub.Register,
		web.Register,
		web.SetLoginRateLimiterCache,
	),
)
