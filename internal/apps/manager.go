package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	// HomelabAppEnableLabel is the label key used to identify homelab applications
	// Container with this label set to "true" will be treated as a homelab app
	HomelabAppEnableLabel = "homelab.app.enable"
)

// Manager manages homelab applications
type Manager struct {
	dockerClient *client.Client
	store        store.Adapter
}

// NewManager creates a new app manager
func NewManager(storeAdapter store.Adapter) (*Manager, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Manager{
		dockerClient: dockerClient,
		store:        storeAdapter,
	}, nil
}

// ScanApps scans Docker containers and filters by homelab app label
func (m *Manager) ScanApps(ctx context.Context) error {
	// List all containers
	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Track found apps to handle removed containers
	foundAppNames := make(map[string]bool)

	// Scan containers with homelab app label
	for _, c := range containers {
		// Check if container has the homelab app enable label set to "true"
		enableLabel, hasLabel := c.Labels[HomelabAppEnableLabel]
		if !hasLabel || enableLabel != "true" {
			continue
		}

		// Get app name from container name (remove leading slash if present)
		appName := ""
		if len(c.Names) > 0 {
			appName = c.Names[0]
			// Remove leading slash from container name
			if len(appName) > 0 && appName[0] == '/' {
				appName = appName[1:]
			}
		}
		if appName == "" {
			// Fallback to container ID if no name
			appName = c.ID[:12]
		}

		foundAppNames[appName] = true

		// Get or create app
		app, err := m.store.GetAppByName(appName)
		if err != nil {
			// Create new app
			app = &model.App{
				Name:   appName,
				Path:   "", // Path is not needed when scanning from Docker
				Status: model.AppStatusUnknown,
			}
			_, err = m.store.CreateApp(app)
			if err != nil {
				flog.Error(fmt.Errorf("failed to create app %s: %w", appName, err))
				continue
			}
		}

		// Associate app with bot by name (one-to-one relationship)
		// Check if bot with same name exists
		bot, err := m.store.GetBotByName(appName)
		if err != nil {
			// Bot not found, log but continue
			flog.Warn("bot not found for app %s: %v", appName, err)
		} else if bot != nil {
			// Bot found, association is established by name
			flog.Debug("app %s associated with bot %s", appName, bot.Name)
		}

		// Update app status from container
		if err := m.updateAppStatusFromContainer(ctx, app, &c); err != nil {
			flog.Error(fmt.Errorf("failed to update app status for %s: %w", appName, err))
			continue
		}

		// Update app in database
		if err := m.store.UpdateApp(app); err != nil {
			flog.Error(fmt.Errorf("failed to update app %s: %w", appName, err))
			continue
		}
	}

	// Mark apps that are no longer found as stopped
	allApps, err := m.store.GetApps()
	if err != nil {
		flog.Error(fmt.Errorf("failed to get all apps: %w", err))
	} else {
		for _, app := range allApps {
			if !foundAppNames[app.Name] {
				// App container not found, mark as stopped
				app.Status = model.AppStatusStopped
				app.ContainerID = ""
				app.DockerInfo = nil
				app.UpdatedAt = time.Now()
				if err := m.store.UpdateApp(app); err != nil {
					flog.Error(fmt.Errorf("failed to update app %s: %w", app.Name, err))
				}
			}
		}
	}

	return nil
}

// updateAppStatusFromContainer updates app status from a specific container
func (m *Manager) updateAppStatusFromContainer(ctx context.Context, app *model.App, c *types.Container) error {
	app.ContainerID = c.ID

	// Get detailed container info
	containerInfo, err := m.dockerClient.ContainerInspect(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Update status based on container state
	switch containerInfo.State.Status {
	case "running":
		app.Status = model.AppStatusRunning
	case "exited", "dead":
		app.Status = model.AppStatusStopped
	case "paused":
		app.Status = model.AppStatusPaused
	case "restarting":
		app.Status = model.AppStatusRestarting
	case "removing":
		app.Status = model.AppStatusRemoving
	default:
		app.Status = model.AppStatusUnknown
	}

	// Store docker info as JSON
	dockerInfo := map[string]interface{}{
		"id":      containerInfo.ID,
		"name":    containerInfo.Name,
		"image":   containerInfo.Config.Image,
		"state":   containerInfo.State.Status,
		"created": containerInfo.Created,
		"ports":   containerInfo.NetworkSettings.Ports,
		"labels":  containerInfo.Config.Labels,
	}

	infoJSON, err := json.Marshal(dockerInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal docker info: %w", err)
	}

	var dockerInfoMap model.JSON
	if err := json.Unmarshal(infoJSON, &dockerInfoMap); err != nil {
		return fmt.Errorf("failed to unmarshal docker info: %w", err)
	}
	app.DockerInfo = dockerInfoMap
	app.UpdatedAt = time.Now()

	return nil
}

// updateAppStatus updates app status by finding container with homelab app label
func (m *Manager) updateAppStatus(ctx context.Context, app *model.App) error {
	// Find container by homelab app label
	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var foundContainer *types.Container
	for _, c := range containers {
		// Check if container has the homelab app enable label set to "true"
		enableLabel, hasLabel := c.Labels[HomelabAppEnableLabel]
		if !hasLabel || enableLabel != "true" {
			continue
		}

		// Get container name and compare with app name
		containerName := ""
		if len(c.Names) > 0 {
			containerName = c.Names[0]
			// Remove leading slash from container name
			if len(containerName) > 0 && containerName[0] == '/' {
				containerName = containerName[1:]
			}
		}

		if containerName == app.Name {
			foundContainer = &c
			break
		}
	}

	if foundContainer == nil {
		app.Status = model.AppStatusStopped
		app.ContainerID = ""
		app.DockerInfo = nil
		return nil
	}

	return m.updateAppStatusFromContainer(ctx, app, foundContainer)
}

// GetAppStatus returns the current status of an app
func (m *Manager) GetAppStatus(ctx context.Context, appName string) (*model.App, error) {
	app, err := m.store.GetAppByName(appName)
	if err != nil {
		return nil, err
	}

	if err := m.updateAppStatus(ctx, app); err != nil {
		return nil, err
	}

	if err := m.store.UpdateApp(app); err != nil {
		return nil, err
	}

	return app, nil
}

// StartApp starts an app's docker containers
func (m *Manager) StartApp(ctx context.Context, appName string) error {
	app, err := m.store.GetAppByName(appName)
	if err != nil {
		return err
	}

	if app.ContainerID == "" {
		return fmt.Errorf("app %s has no container", appName)
	}

	err = m.dockerClient.ContainerStart(ctx, app.ContainerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update status
	if err := m.updateAppStatus(ctx, app); err != nil {
		return err
	}

	return m.store.UpdateApp(app)
}

// StopApp stops an app's docker containers
func (m *Manager) StopApp(ctx context.Context, appName string) error {
	app, err := m.store.GetAppByName(appName)
	if err != nil {
		return err
	}

	if app.ContainerID == "" {
		return fmt.Errorf("app %s has no container", appName)
	}

	timeoutSeconds := 10
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = m.dockerClient.ContainerStop(stopCtx, app.ContainerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update status
	if err := m.updateAppStatus(ctx, app); err != nil {
		return err
	}

	return m.store.UpdateApp(app)
}

// RestartApp restarts an app's docker containers
func (m *Manager) RestartApp(ctx context.Context, appName string) error {
	app, err := m.store.GetAppByName(appName)
	if err != nil {
		return err
	}

	if app.ContainerID == "" {
		return fmt.Errorf("app %s has no container", appName)
	}

	timeoutSeconds := 10
	restartCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = m.dockerClient.ContainerRestart(restartCtx, app.ContainerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// Update status
	if err := m.updateAppStatus(ctx, app); err != nil {
		return err
	}

	return m.store.UpdateApp(app)
}

// GetAppWithBot returns app with associated bot (by name)
func (m *Manager) GetAppWithBot(ctx context.Context, appName string) (*model.App, *model.Bot, error) {
	app, err := m.store.GetAppByName(appName)
	if err != nil {
		return nil, nil, err
	}

	// Get associated bot by name (one-to-one relationship)
	bot, err := m.store.GetBotByName(appName)
	if err != nil {
		// Bot not found, return app without bot
		return app, nil, nil
	}

	return app, bot, nil
}
