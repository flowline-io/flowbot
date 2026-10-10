package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/homelab/probe"
	"github.com/flowline-io/flowbot/pkg/types"
)

var homelabRuntime homelab.Runtime = homelab.NoopRuntime{}

// RunHomelabScan executes a full homelab scan + probe + registry update cycle.
// It walks the configured apps directory, discovers compose files, runs the
// probe engine for endpoint/auth discovery, and replaces the default registry.
// Exported for use by the homelab web handler to support manual rescan.
func RunHomelabScan(cfg config.Homelab) error {
	homeConfig := homelabConfig(cfg)
	if homeConfig.AppsDir == "" && homeConfig.Root == "" {
		return errors.New("homelab app registry disabled: apps_dir and root are empty")
	}
	apps, err := homelab.NewScanner(homeConfig).Scan()
	if err != nil {
		return fmt.Errorf("scan homelab apps: %w", err)
	}

	if eng := probe.NewEngine(homeConfig.Discovery); eng != nil {
		ctx, cancel := context.WithTimeout(context.Background(), homeConfig.Discovery.ProbeTimeout*2)
		defer cancel()
		probeResults := eng.ProbeAll(ctx, apps)
		if len(probeResults) > 0 {
			apps = mergeProbeResults(apps, probeResults)
		}
	}

	homelab.DefaultRegistry.Replace(apps)
	homelab.DefaultRegistry.SetPermissions(homeConfig.Permissions)
	if store.Database != nil && store.Database.GetClient() != nil {
		if err := store.NewHubStore(store.Database.GetClient()).SaveHomelabApps(context.Background(), apps); err != nil {
			return fmt.Errorf("persist homelab apps: %w", err)
		}
	}
	flog.Info("homelab app registry rescanned with %d apps", len(apps))
	return nil
}

func initHomelabRegistry(cfg config.Homelab) error {
	hcfg := homelabConfig(cfg)
	homelabRuntime = homelab.NewRuntime(hcfg.Runtime, hcfg.AppsDir)
	homelab.DefaultRuntime = homelabRuntime
	homelab.SetRunRescan(func() error { return RunHomelabScan(cfg) })
	if cfg.AppsDir == "" && cfg.Root == "" {
		flog.Info("homelab app registry disabled: homelab.apps_dir and homelab.root are empty")
		return nil
	}
	return RunHomelabScan(cfg)
}

func homelabConfig(cfg config.Homelab) homelab.Config {
	permissions := homelab.Permissions{
		Status:  cfg.Permissions.Status,
		Logs:    cfg.Permissions.Logs,
		Start:   cfg.Permissions.Start,
		Stop:    cfg.Permissions.Stop,
		Restart: cfg.Permissions.Restart,
		Pull:    cfg.Permissions.Pull,
		Update:  cfg.Permissions.Update,
		Exec:    cfg.Permissions.Exec,
	}
	discovery := homelab.DiscoveryConfig{
		ProbeEnabled:       cfg.Discovery.ProbeEnabled,
		ProbeConcurrency:   cfg.Discovery.ProbeConcurrency,
		FingerprintEnabled: cfg.Discovery.FingerprintEnabled,
		LabelPriority:      cfg.Discovery.LabelPriority,
	}
	if cfg.Discovery.ProbeTimeout != "" {
		if d, err := time.ParseDuration(cfg.Discovery.ProbeTimeout); err == nil {
			discovery.ProbeTimeout = d
		}
	}
	if discovery.ProbeTimeout == 0 {
		discovery.ProbeTimeout = 5 * time.Second
	}
	if discovery.ProbeConcurrency <= 0 {
		discovery.ProbeConcurrency = 4
	}
	return homelab.Config{
		Root:        cfg.Root,
		AppsDir:     cfg.AppsDir,
		ComposeFile: cfg.ComposeFile,
		Allowlist:   cfg.Allowlist,
		Runtime: homelab.RuntimeConfig{
			Mode:         homelab.RuntimeMode(cfg.Runtime.Mode),
			DockerSocket: cfg.Runtime.DockerSocket,
			SSHHost:      cfg.Runtime.SSHHost,
			SSHPort:      cfg.Runtime.SSHPort,
			SSHUser:      cfg.Runtime.SSHUser,
			SSHPassword:  cfg.Runtime.SSHPassword,
			SSHKey:       cfg.Runtime.SSHKey,
			SSHHostKey:   cfg.Runtime.SSHHostKey,
		},
		Permissions: permissions,
		Discovery:   discovery,
	}
}

// mergeProbeResults enriches apps with capabilities discovered by the probe
// engine. Probe results are matched to apps by name. When label_priority is
// true, existing label-derived capabilities are preserved and probe data only
// fills in missing endpoint/auth information.
func mergeProbeResults(apps []homelab.App, probeResults []probe.ProbeResult) []homelab.App {
	probeByApp := make(map[string][]homelab.AppCapability, len(probeResults))
	for _, pr := range probeResults {
		probeByApp[pr.AppName] = pr.Capabilities
	}
	for i := range apps {
		probeCaps, ok := probeByApp[apps[i].Name]
		if !ok {
			continue
		}
		if len(apps[i].Capabilities) == 0 {
			apps[i].Capabilities = probeCaps
			continue
		}
		// Enrich existing capabilities with probe data.
		for _, probeCap := range probeCaps {
			for j := range apps[i].Capabilities {
				if apps[i].Capabilities[j].Endpoint == nil && probeCap.Endpoint != nil {
					apps[i].Capabilities[j].Endpoint = probeCap.Endpoint
				}
				if apps[i].Capabilities[j].Auth == nil && probeCap.Auth != nil {
					apps[i].Capabilities[j].Auth = probeCap.Auth
				}
			}
		}
	}
	return apps
}

func startHomelabImageCheckLoop(lc fx.Lifecycle) {
	interval, ok := homelab.ImageCheckInterval(
		homelab.RuntimeMode(config.App.Homelab.Runtime.Mode),
		config.App.Homelab.ImageCheck.Interval,
	)
	if !ok {
		flog.Info("homelab image check disabled")
		return
	}
	stop := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go homelabImageCheckLoop(stop, interval, runHomelabImageCheck)
			flog.Info("homelab image check started (interval=%s)", interval)
			return nil
		},
		OnStop: func(_ context.Context) error {
			close(stop)
			return nil
		},
	})
}

func homelabImageCheckLoop(stop <-chan struct{}, interval time.Duration, run func()) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			run()
			timer.Reset(interval)
		}
	}
}

func runHomelabImageCheck() {
	ctx := context.Background()
	updates := homelab.CheckImageUpdates(ctx, homelabRuntime, homelab.DefaultRegistry.List())
	if len(updates) == 0 {
		return
	}
	if store.Database == nil || store.Database.GetClient() == nil {
		flog.Warn("homelab image check: skipped emit, store not ready")
		return
	}
	eventStore := store.EventStoreFromDB()
	publishHomelabImageUpdates(ctx, updates, eventStore.DataEventExists, func(ctx context.Context, de types.DataEvent) error {
		return persistAndPublishDataEvent(ctx, eventStore, event.PublishMessage, DataEventTopic, de, "homelab_image_check")
	})
}

func publishHomelabImageUpdates(
	ctx context.Context,
	updates []homelab.ImageUpdate,
	exists func(context.Context, string, string) (bool, error),
	emit func(context.Context, types.DataEvent) error,
) {
	for _, u := range updates {
		key := homelab.ImageUpdateIdempotencyKey(u.AppName, u.Service, u.RemoteDigest)
		found, err := exists(ctx, types.EventHomelabImageUpdateAvailable, key)
		if err != nil {
			flog.Warn("homelab image check: exists %s: %v", key, err)
			continue
		}
		if found {
			continue
		}
		if err := emit(ctx, dataEventFromImageUpdate(u)); err != nil {
			flog.Warn("homelab image check: emit %s: %v", key, err)
		}
	}
}

func dataEventFromImageUpdate(u homelab.ImageUpdate) types.DataEvent {
	return types.DataEvent{
		EventID:        types.Id(),
		EventType:      types.EventHomelabImageUpdateAvailable,
		Source:         homelab.ImageCheckSource,
		App:            u.AppName,
		Capability:     u.Capability,
		EntityID:       homelab.ImageUpdateEntityID(u.AppName, u.Service),
		IdempotencyKey: homelab.ImageUpdateIdempotencyKey(u.AppName, u.Service, u.RemoteDigest),
		CreatedAt:      time.Now(),
		Data: types.KV{
			"image":          u.Image,
			"tag":            u.Tag,
			"current_digest": u.CurrentDigest,
			"remote_digest":  u.RemoteDigest,
		},
	}
}
