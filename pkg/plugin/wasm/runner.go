// Package wasm provides the WebAssembly-based plugin runner using wazero.
package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/flowline-io/flowbot/pkg/plugin"
)

// WasmRunner implements plugin.Runner using wazero for in-process wasm execution.
type WasmRunner struct {
	manifest     *plugin.Manifest
	runtime      wazero.Runtime
	compiled     wazero.CompiledModule
	instance     atomic.Pointer[api.Module]
	hostBindings *HostBindings
	timeout      time.Duration
	memMax       uint32

	pool        sync.Pool
	poolSize    int
	poolTimeout time.Duration

	inflight sync.WaitGroup
	started  atomic.Bool
}

// NewWasmRunner creates a WasmRunner for the given plugin manifest.
func NewWasmRunner(m *plugin.Manifest) (*WasmRunner, error) {
	wasmCfg := m.Wasm
	if wasmCfg == nil {
		return nil, fmt.Errorf("wasm config required")
	}

	timeout := 30 * time.Second
	if wasmCfg.Permissions != nil && wasmCfg.Permissions.Execution != nil {
		if d, err := time.ParseDuration(wasmCfg.Permissions.Execution.Timeout); err == nil {
			timeout = d
		}
	}

	memMax := uint32(64 * 1024 * 1024)
	if wasmCfg.Permissions != nil && wasmCfg.Permissions.Memory != nil {
		if s := wasmCfg.Permissions.Memory.Max; s != "" {
			if parsed, err := parseMemBytes(s); err == nil {
				memMax = parsed
			}
		}
	}

	poolSize := 4
	poolTimeout := 5 * time.Second
	if wasmCfg.Pool != nil {
		if wasmCfg.Pool.MaxInstances > 0 {
			poolSize = wasmCfg.Pool.MaxInstances
		}
		if wasmCfg.Pool.WaitTimeout != "" {
			if d, err := time.ParseDuration(wasmCfg.Pool.WaitTimeout); err == nil {
				poolTimeout = d
			}
		}
	}

	bindings := &HostBindings{}
	if wasmCfg.Permissions != nil {
		bindings.httpPerms = buildAllowlist(wasmCfg.Permissions)
	}

	rt := wazero.NewRuntime(context.Background())

	return &WasmRunner{
		manifest:     m,
		runtime:      rt,
		hostBindings: bindings,
		timeout:      timeout,
		memMax:       memMax,
		poolSize:     poolSize,
		poolTimeout:  poolTimeout,
	}, nil
}

// Load compiles the wasm module, registers host functions, and creates the primary instance.
func (r *WasmRunner) Load(ctx context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	if m.Wasm == nil {
		return nil, fmt.Errorf("wasm load: no wasm config")
	}

	// Install WASI snapshot preview1
	wasi.MustInstantiate(ctx, r.runtime)

	// Register host functions under the "flowbot" module namespace
	if err := r.hostBindings.exportToRuntime(ctx, r.runtime); err != nil {
		return nil, fmt.Errorf("wasm load: host bindings: %w", err)
	}

	wasmBytes, err := os.ReadFile(m.Wasm.Module)
	if err != nil {
		return nil, fmt.Errorf("wasm load: read module %s: %w", m.Wasm.Module, err)
	}

	compiled, err := r.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("wasm load: compile: %w", err)
	}
	r.compiled = compiled

	// Pre-configure the instance pool with lazy instantiation
	r.pool.New = func() any {
		mod, err := r.runtime.InstantiateModule(context.Background(), r.compiled,
			r.moduleConfig().WithName(""))
		if err != nil {
			return nil
		}
		return mod
	}

	// Create the primary instance (named, for debugging)
	mod, err := r.runtime.InstantiateModule(ctx, compiled, r.moduleConfig())
	if err != nil {
		return nil, fmt.Errorf("wasm load: instantiate: %w", err)
	}
	r.instance.Store(&mod)

	return &plugin.PluginInfo{
		Name:         m.Name,
		Version:      m.Version,
		Provides:     m.Provides,
		ConfigSchema: m.ConfigSchema,
	}, nil
}

// Start calls the wasm init() export with the plugin configuration.
func (r *WasmRunner) Start(ctx context.Context, config json.RawMessage) error {
	mod := *r.instance.Load()

	ptr, size, err := writeJSON(ctx, mod, config)
	if err != nil {
		return fmt.Errorf("wasm start: write config: %w", err)
	}

	fn := mod.ExportedFunction("init")
	if fn != nil {
		if _, err := fn.Call(ctx, uint64(ptr), uint64(size)); err != nil {
			return fmt.Errorf("wasm start init: %w", err)
		}
	}
	r.started.Store(true)
	return nil
}

// Stop drains in-flight calls and closes all wasm instances.
func (r *WasmRunner) Stop(ctx context.Context) error {
	r.started.Store(false)

	done := make(chan struct{})
	go func() {
		r.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}

	if mod := r.instance.Load(); mod != nil {
		(*mod).Close(ctx)
	}
	if r.compiled != nil {
		r.compiled.Close(ctx)
	}
	return r.runtime.Close(ctx)
}

// Call invokes a named wasm export with JSON parameters and returns JSON result.
func (r *WasmRunner) Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error) {
	if !r.started.Load() {
		return nil, fmt.Errorf("wasm call: plugin not started")
	}

	r.inflight.Add(1)
	defer r.inflight.Done()

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	mod := r.acquireInstance()
	if mod == nil {
		return nil, fmt.Errorf("wasm call: no available instance")
	}
	defer r.releaseInstance(mod)

	fn := mod.ExportedFunction(function)
	if fn == nil {
		return nil, fmt.Errorf("wasm call: export %q not found", function)
	}

	ptr, size, err := writeBytes(ctx, mod, params)
	if err != nil {
		return nil, fmt.Errorf("wasm call: write params: %w", err)
	}

	results, err := fn.Call(ctx, uint64(ptr), uint64(size))
	if err != nil {
		return nil, fmt.Errorf("wasm call %q: %w", function, err)
	}

	var raw json.RawMessage
	if err := readJSON(ctx, mod, results[0], &raw); err != nil {
		return nil, fmt.Errorf("wasm call %q: read result: %w", function, err)
	}
	return raw, nil
}

// Health checks the plugin's is_ready export.
func (r *WasmRunner) Health(ctx context.Context) (*plugin.HealthStatus, error) {
	mod := r.acquireInstance()
	if mod == nil {
		return &plugin.HealthStatus{Ready: false, LastError: "no instance available"}, nil
	}
	defer r.releaseInstance(mod)

	fn := mod.ExportedFunction("is_ready")
	if fn == nil {
		return &plugin.HealthStatus{Ready: false, LastError: "is_ready export not found"}, nil
	}

	results, err := fn.Call(ctx)
	if err != nil {
		return &plugin.HealthStatus{Ready: false, LastError: err.Error()}, nil
	}

	ready := false
	if len(results) > 0 && results[0] == 1 {
		ready = true
	}
	return &plugin.HealthStatus{Ready: ready}, nil
}

// moduleConfig returns a sandboxed ModuleConfig for wasm instantiation.
func (r *WasmRunner) moduleConfig() wazero.ModuleConfig {
	cfg := wazero.NewModuleConfig().
		WithSysNanotime().
		WithSysWalltime().
		WithStartFunctions() // No auto-start; we call exports manually

	if r.manifest.Wasm.Permissions != nil {
		for _, fs := range r.manifest.Wasm.Permissions.Filesystem {
			if fs.Mode == "read" || fs.Mode == "readwrite" {
				cfg = cfg.WithFS(os.DirFS(fs.Path))
			}
		}
	}

	return cfg
}

func (r *WasmRunner) acquireInstance() api.Module {
	if obj := r.pool.Get(); obj != nil {
		if mod, ok := obj.(api.Module); ok && mod != nil {
			return mod
		}
	}
	if inst := r.instance.Load(); inst != nil {
		return *inst
	}
	return nil
}

func (r *WasmRunner) releaseInstance(mod api.Module) {
	r.pool.Put(mod)
}

// parseMemBytes parses a memory size string like "64MB", "128KB", "2GB" into bytes.
func parseMemBytes(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory size")
	}

	mult := uint64(1)
	switch {
	case strings.HasSuffix(s, "GB"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		mult = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}

	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse memory size %q: %w", s, err)
	}
	return uint32(v * mult), nil
}
