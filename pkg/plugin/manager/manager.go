// Package manager provides the PluginManager that orchestrates plugin
// discovery, loading, lifecycle, and hot-reload.
package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/plugin/adapter"
	"github.com/flowline-io/flowbot/pkg/plugin/grpc"
	"github.com/flowline-io/flowbot/pkg/plugin/source"
	"github.com/flowline-io/flowbot/pkg/plugin/wasm"
	"github.com/flowline-io/flowbot/pkg/providers"
)

// PluginManager orchestrates plugin discovery, loading, lifecycle, and hot-reload.
type PluginManager struct {
	instances map[string]*PluginInstance
	config    *PluginConfig
	mu        sync.RWMutex
	logger    zerolog.Logger
}

// PluginInstance tracks a loaded plugin's state.
type PluginInstance struct {
	Identity  string
	Manifest  *plugin.Manifest
	Runner    plugin.Runner
	Adapters  *PluginAdapters
	State     plugin.PluginState
	StartedAt time.Time
	LastError error
}

// PluginAdapters holds all adapters for a plugin instance.
type PluginAdapters struct {
	Module    *adapter.PluginModuleAdapter
	Abilities []*adapter.PluginAbilityAdapter
	Provider  *adapter.PluginProviderAdapter
}

// NewPluginManager creates a PluginManager from configuration.
func NewPluginManager(cfg *PluginConfig, log zerolog.Logger) *PluginManager {
	return &PluginManager{
		instances: make(map[string]*PluginInstance),
		config:    cfg,
		logger:    log.With().Str("component", "plugin-manager").Logger(),
	}
}

// Init discovers and loads all plugins from configured sources.
func (m *PluginManager) Init(ctx context.Context, pluginConfigs map[string]json.RawMessage) error {
	if m.config == nil || !m.config.Enabled {
		m.logger.Info().Msg("plugin system disabled")
		return nil
	}
	if m.config.MaxPlugins > 0 && len(m.instances) >= m.config.MaxPlugins {
		m.logger.Warn().Int("max", m.config.MaxPlugins).Msg("max plugins reached, skipping discovery")
		return nil
	}

	for _, srcCfg := range m.config.Sources {
		src, err := source.NewSource(srcCfg)
		if err != nil {
			m.logger.Error().Err(err).Str("type", srcCfg.Type).Msg("failed to create source")
			continue
		}

		manifests, err := src.Discover(ctx)
		if err != nil {
			m.logger.Error().Err(err).Str("type", srcCfg.Type).Msg("discovery failed")
			continue
		}

		for _, manifest := range manifests {
			identity := deriveIdentity(srcCfg, manifest)
			if _, exists := m.instances[identity]; exists {
				m.logger.Warn().Str("identity", identity).Msg("duplicate plugin identity, skipping")
				continue
			}
			cfg := pluginConfigs[identity]
			if m.config.MaxPlugins > 0 && len(m.instances) >= m.config.MaxPlugins {
				m.logger.Warn().Int("max", m.config.MaxPlugins).Msg("max plugins reached, stopping discovery")
				return nil
			}
			if err := m.loadPlugin(ctx, identity, manifest, cfg); err != nil {
				m.logger.Error().Err(err).Str("identity", identity).Msg("failed to load plugin")
			}
		}
	}
	return nil
}

func deriveIdentity(srcCfg SourceConfig, manifest *plugin.Manifest) string {
	switch srcCfg.Type {
	case "local", "oci", "git":
		return manifest.Name
	default:
		return manifest.Name
	}
}

// loadPlugin loads a single plugin, creates adapters, and registers them.
func (m *PluginManager) loadPlugin(ctx context.Context, identity string, manifest *plugin.Manifest, cfg json.RawMessage) error {
	m.logger.Info().Str("identity", identity).Str("runtime", string(manifest.Runtime)).Msg("loading plugin")

	if err := manifest.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	var runner plugin.Runner
	var err error
	switch manifest.Runtime {
	case plugin.RuntimeGRPC:
		runner, err = grpc.NewGrpcRunner(manifest)
	case plugin.RuntimeWasm:
		runner, err = wasm.NewWasmRunner(manifest)
	default:
		return fmt.Errorf("unknown runtime: %s", manifest.Runtime)
	}
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}

	if _, err := runner.Load(ctx, manifest); err != nil {
		return fmt.Errorf("runner load: %w", err)
	}

	adapters := &PluginAdapters{}
	if manifest.Provides.Module {
		adapters.Module = adapter.NewModuleAdapter(manifest, runner)
		module.Register(identity, adapters.Module)
	}
	for _, ab := range manifest.Provides.Abilities {
		a := adapter.NewAbilityAdapter(runner, ab.Capability, ab.Operations)
		if err := a.Register(); err != nil {
			return fmt.Errorf("register ability %s: %w", ab.Capability, err)
		}
		adapters.Abilities = append(adapters.Abilities, a)
	}
	if manifest.Provides.Provider != nil {
		adapters.Provider = adapter.NewProviderAdapter(runner, manifest.Provides.Provider.Name)
		if manifest.Provides.Provider.OAuth {
			providers.RegisterOAuthProvider(manifest.Provides.Provider.Name, func() providers.OAuthProvider { return adapters.Provider })
		}
	}

	if err := runner.Start(ctx, cfg); err != nil {
		return fmt.Errorf("runner start: %w", err)
	}

	m.mu.Lock()
	m.instances[identity] = &PluginInstance{
		Identity:  identity,
		Manifest:  manifest,
		Runner:    runner,
		Adapters:  adapters,
		State:     plugin.StateRunning,
		StartedAt: time.Now(),
	}
	m.mu.Unlock()

	m.logger.Info().Str("identity", identity).Msg("plugin loaded")
	return nil
}

// ReloadPlugin hot-reloads a plugin with validate-then-swap semantics.
func (m *PluginManager) ReloadPlugin(ctx context.Context, identity string, newManifest *plugin.Manifest, newCfg json.RawMessage) error {
	m.mu.RLock()
	old := m.instances[identity]
	m.mu.RUnlock()
	if old == nil {
		return fmt.Errorf("plugin %s not found", identity)
	}

	if err := newManifest.ValidateConfig(newCfg); err != nil {
		return fmt.Errorf("hot-reload aborted: config validation: %w", err)
	}

	var newRunner plugin.Runner
	var err error
	switch newManifest.Runtime {
	case plugin.RuntimeGRPC:
		newRunner, err = grpc.NewGrpcRunner(newManifest)
	case plugin.RuntimeWasm:
		newRunner, err = wasm.NewWasmRunner(newManifest)
	}
	if err != nil {
		return fmt.Errorf("create new runner: %w", err)
	}

	if _, err := newRunner.Load(ctx, newManifest); err != nil {
		return fmt.Errorf("load new runner: %w", err)
	}
	if err := newRunner.Start(ctx, newCfg); err != nil {
		return fmt.Errorf("start new runner: %w", err)
	}

	providesChanged := !providesEqual(old.Manifest.Provides, newManifest.Provides)
	if providesChanged {
		unregisterAdapters(old)
		registerAdapters(identity, newManifest, newRunner)
	} else if old.Adapters.Module != nil {
		old.Adapters.Module.SwapRunner(newRunner)
	}

	drainCtx, cancel := context.WithTimeout(ctx, m.config.DrainTimeout)
	defer cancel()
	old.Runner.Stop(drainCtx)

	m.mu.Lock()
	m.instances[identity] = &PluginInstance{
		Identity:  identity,
		Manifest:  newManifest,
		Runner:    newRunner,
		Adapters:  old.Adapters,
		State:     plugin.StateRunning,
		StartedAt: time.Now(),
	}
	m.mu.Unlock()

	return nil
}

// UnloadPlugin unloads a plugin and removes it from all registries.
func (m *PluginManager) UnloadPlugin(ctx context.Context, identity string) error {
	m.mu.Lock()
	inst, ok := m.instances[identity]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("plugin %s not found", identity)
	}
	delete(m.instances, identity)
	m.mu.Unlock()

	unregisterAdapters(inst)

	drainCtx, cancel := context.WithTimeout(ctx, m.config.DrainTimeout)
	defer cancel()
	return inst.Runner.Stop(drainCtx)
}

// List returns all loaded plugin instances.
func (m *PluginManager) List() []*PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*PluginInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		result = append(result, inst)
	}
	return result
}

func unregisterAdapters(inst *PluginInstance) {
	if inst.Adapters.Module != nil {
		module.Unregister(inst.Identity)
	}
	for _, a := range inst.Adapters.Abilities {
		a.Unregister()
	}
	if inst.Adapters.Provider != nil && inst.Manifest.Provides.Provider != nil && inst.Manifest.Provides.Provider.OAuth {
		providers.UnregisterOAuthProvider(inst.Manifest.Provides.Provider.Name)
	}
}

func registerAdapters(identity string, manifest *plugin.Manifest, runner plugin.Runner) {
	if manifest.Provides.Module {
		modAdapter := adapter.NewModuleAdapter(manifest, runner)
		module.Register(identity, modAdapter)
	}
	for _, ab := range manifest.Provides.Abilities {
		a := adapter.NewAbilityAdapter(runner, ab.Capability, ab.Operations)
		a.Register()
	}
	if manifest.Provides.Provider != nil && manifest.Provides.Provider.OAuth {
		provAdapter := adapter.NewProviderAdapter(runner, manifest.Provides.Provider.Name)
		providers.RegisterOAuthProvider(manifest.Provides.Provider.Name, func() providers.OAuthProvider { return provAdapter })
	}
}

// providesEqual checks if two Provides declarations are equivalent.
func providesEqual(a, b plugin.Provides) bool {
	if a.Module != b.Module {
		return false
	}
	if (a.Provider == nil) != (b.Provider == nil) {
		return false
	}
	if a.Provider != nil && b.Provider != nil {
		if a.Provider.Name != b.Provider.Name || a.Provider.OAuth != b.Provider.OAuth {
			return false
		}
	}
	if len(a.Abilities) != len(b.Abilities) {
		return false
	}
	for i := range a.Abilities {
		if a.Abilities[i].Capability != b.Abilities[i].Capability {
			return false
		}
		if !stringSlicesEqual(a.Abilities[i].Operations, b.Abilities[i].Operations) {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
