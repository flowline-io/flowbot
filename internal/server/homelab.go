package server

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/homelab/probe"
)

var homelabRuntime homelab.Runtime = homelab.NoopRuntime{}

func initHomelabRegistry(cfg config.Homelab) error {
	homeConfig := homelabConfig(cfg)
	homelabRuntime = homelab.NewRuntime(homeConfig.Runtime, homeConfig.AppsDir)
	homelab.DefaultRuntime = homelabRuntime
	if homeConfig.AppsDir == "" {
		flog.Info("homelab app registry disabled: homelab.apps_dir is empty")
		return nil
	}
	apps, err := homelab.NewScanner(homeConfig).Scan()
	if err != nil {
		return fmt.Errorf("scan homelab apps: %w", err)
	}

	// Run probe engine to enrich apps with runtime endpoint and auth discovery.
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
	if store.Database != nil && store.Database.GetDB() != nil {
		if err := store.NewHubStore(store.Database.GetDB()).SaveHomelabApps(apps); err != nil {
			return fmt.Errorf("persist homelab apps: %w", err)
		}
	}
	flog.Info("homelab app registry initialized with %d apps", len(apps))
	return nil
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
		ProbeNetworks:      cfg.Discovery.ProbeNetworks,
		ProbePortStrategy:  cfg.Discovery.ProbePortStrategy,
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
	if discovery.ProbePortStrategy == "" {
		discovery.ProbePortStrategy = "published"
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
