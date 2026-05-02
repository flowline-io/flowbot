package server

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
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
	}
}
