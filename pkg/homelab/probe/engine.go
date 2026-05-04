package probe

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

// Engine orchestrates runtime probing of homelab apps to discover API endpoints
// and authentication mechanisms.
type Engine struct {
	config homelab.DiscoveryConfig
	probe  *HTTPProbe
}

// NewEngine creates a probe Engine with the given discovery configuration.
// Returns nil if probing is disabled.
func NewEngine(config homelab.DiscoveryConfig) *Engine {
	if !config.ProbeEnabled {
		return nil
	}
	return &Engine{
		config: config,
		probe:  NewHTTPProbe(config.ProbeTimeout),
	}
}

// ProbeAll runs probes against all provided apps and returns enriched
// capabilities grouped by app name. Apps without running status are skipped.
func (e *Engine) ProbeAll(ctx context.Context, apps []homelab.App) []ProbeResult {
	if e == nil || e.probe == nil {
		return nil
	}

	concurrency := e.config.ProbeConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	type task struct {
		app    homelab.App
		target string
	}
	type appResult struct {
		appName      string
		capabilities []homelab.AppCapability
	}
	tasks := make(chan task)
	results := make(chan appResult, len(apps))

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				caps := e.probeApp(ctx, t.app, t.target)
				if len(caps) > 0 {
					results <- appResult{appName: t.app.Name, capabilities: caps}
				}
			}
		}()
	}

	go func() {
		for _, app := range apps {
			if app.Status != homelab.AppStatusRunning {
				continue
			}
			targets := e.resolveTargets(app)
			for _, target := range targets {
				select {
				case tasks <- task{app: app, target: target}:
				case <-ctx.Done():
					close(tasks)
					return
				}
			}
		}
		close(tasks)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Group results by app name.
	grouped := make(map[string][]homelab.AppCapability)
	for r := range results {
		grouped[r.appName] = append(grouped[r.appName], r.capabilities...)
	}

	var all []ProbeResult
	for appName, caps := range grouped {
		all = append(all, ProbeResult{
			AppName:      appName,
			Capabilities: deduplicateCapabilities(caps),
		})
	}
	return all
}

// resolveTargets figures out which URLs to probe for a given app, based on
// its port mappings and the configured port strategy. Both http and https
// schemes are attempted for each target.
func (e *Engine) resolveTargets(app homelab.App) []string {
	var targets []string
	for _, port := range app.Ports {
		if port.Protocol != "tcp" {
			continue
		}
		host := e.resolveHost(port)
		if host == "" {
			continue
		}
		hostPort := e.resolveHostPort(port)
		if hostPort == "" {
			continue
		}
		address := net.JoinHostPort(host, hostPort)
		targets = append(targets, fmt.Sprintf("http://%s", address), fmt.Sprintf("https://%s", address))
	}
	return targets
}

func (e *Engine) resolveHost(port homelab.PortMapping) string {
	if port.Host != "" {
		return port.Host
	}
	if port.HostPort != "" {
		return "localhost"
	}
	return ""
}

func (e *Engine) resolveHostPort(port homelab.PortMapping) string {
	switch e.config.ProbePortStrategy {
	case "container":
		if port.Container != "" {
			return port.Container
		}
		return port.HostPort
	case "both":
		if port.HostPort != "" {
			return port.HostPort
		}
		return port.Container
	default:
		if port.HostPort != "" {
			return port.HostPort
		}
		return port.Container
	}
}

// probeApp probes a single target URL and returns discovered capabilities.
func (e *Engine) probeApp(ctx context.Context, app homelab.App, target string) []homelab.AppCapability {
	result := e.probe.ProbeEndpoint(ctx, target)
	if result == nil {
		return nil
	}

	// Combine with existing label-derived capabilities.
	if len(app.Capabilities) > 0 {
		var enriched []homelab.AppCapability
		for _, existing := range app.Capabilities {
			en := existing
			if en.Endpoint == nil && result.BaseURL != "" {
				en.Endpoint = &homelab.EndpointInfo{
					BaseURL: result.BaseURL,
					Health:  result.HealthURL,
				}
			}
			if en.Auth == nil && result.Auth != nil {
				en.Auth = result.Auth
			}
			enriched = append(enriched, en)
		}
		flog.Info("homelab probe: app=%s target=%s auth=%s health=%s (labels enriched)",
			app.Name, target, authTypeLabel(result.Auth), result.HealthURL)
		return enriched
	}

	// No label capabilities; use fingerprint matches.
	var caps []homelab.AppCapability
	for _, match := range result.Matches {
		if match.Confidence > 0 {
			caps = append(caps, match.Capability)
		}
	}

	// If no fingerprint match, still report raw endpoint info.
	if len(caps) == 0 {
		caps = append(caps, homelab.AppCapability{
			Endpoint: &homelab.EndpointInfo{
				BaseURL: result.BaseURL,
				Health:  result.HealthURL,
			},
			Auth: result.Auth,
		})
	}

	flog.Info("homelab probe: app=%s target=%s auth=%s health=%s",
		app.Name, target, authTypeLabel(result.Auth), result.HealthURL)
	return caps
}

func deduplicateCapabilities(caps []homelab.AppCapability) []homelab.AppCapability {
	seen := make(map[string]bool)
	var result []homelab.AppCapability
	for _, mc := range caps {
		key := mc.Capability + "|" + mc.Backend
		if !seen[key] {
			seen[key] = true
			result = append(result, mc)
		}
	}
	return result
}

func authTypeLabel(auth *homelab.AuthInfo) string {
	if auth == nil {
		return "unknown"
	}
	return strings.ToLower(string(auth.Type))
}
