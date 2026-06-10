// Package executor provides the pipeline execution engine.
package executor

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	capabilityruntime "github.com/flowline-io/flowbot/pkg/executor/runtime/capability"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/docker"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/machine"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
	"github.com/flowline-io/flowbot/pkg/types"
)

// engine state constants used with atomic operations.
const (
	stateIdle int32 = iota
	stateRunning
	stateClosed
)

// Engine manages the lifecycle of a task runtime, enforcing single-task
// execution and providing state introspection.
type Engine struct {
	state       atomic.Int32
	mounters    map[string]*runtime.MultiMounter
	runtime     runtime.Runtime
	limits      Limits
	runtimeType string
}

// Limits defines default resource constraints applied to tasks that do not
// specify their own.
type Limits struct {
	DefaultCPUsLimit   string
	DefaultMemoryLimit string
}

// New creates a new Engine for the given runtime type without initializing
// the underlying runtime. The runtime is lazily initialized on the first Run.
func New(runtimeType string) *Engine {
	e := &Engine{
		mounters:    make(map[string]*runtime.MultiMounter),
		runtimeType: runtimeType,
	}
	// stateIdle is zero value; atomic.Int32 starts at 0.
	return e
}

// State returns the current engine state as a human-readable string.
func (e *Engine) State() string {
	switch e.state.Load() {
	case stateRunning:
		return "RUNNING"
	case stateClosed:
		return "CLOSED"
	default:
		return "IDLE"
	}
}

// Close cleans up the engine's runtime resources and marks the engine as closed.
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (e *Engine) Close() error {
	if !e.state.CompareAndSwap(stateIdle, stateClosed) &&
		!e.state.CompareAndSwap(stateRunning, stateClosed) {
		return nil // already closed
	}
	if e.runtime != nil {
		err := e.runtime.Close()
		e.runtime = nil
		return err
	}
	return nil
}

// Run executes the given task, enforcing that only one task runs at a time.
// It returns an error if the engine is already running or has been closed.
func (e *Engine) Run(ctx context.Context, t *types.Task) error {
	if !e.state.CompareAndSwap(stateIdle, stateRunning) {
		return fmt.Errorf("engine is not idle, current state: %s", e.State())
	}
	defer e.state.Store(stateIdle)
	return e.runTask(ctx, t)
}

func (e *Engine) runTask(ctx context.Context, t *types.Task) error {
	if _, err := e.initRuntime(); err != nil {
		return err
	}

	return e.doRunTask(ctx, t)
}

// initRuntime lazily initializes the runtime on first call. Subsequent calls
// return the already-initialized runtime without re-creating it.
func (e *Engine) initRuntime() (runtime.Runtime, error) {
	if e.runtime != nil {
		return e.runtime, nil
	}
	e.limits = Limits{
		DefaultCPUsLimit:   config.App.Executor.Limits.Cpus,
		DefaultMemoryLimit: config.App.Executor.Limits.Memory,
	}
	switch e.runtimeType {
	case runtime.Docker:
		mounter, ok := e.mounters[runtime.Docker]
		if !ok {
			mounter = runtime.NewMultiMounter()
		}
		// register bind mounter
		bm := docker.NewBindMounter(docker.BindConfig{
			Allowed: config.App.Executor.Mounts.Bind.Allowed,
		})
		mounter.RegisterMounter("bind", bm)
		// register volume mounter
		vm, err := docker.NewVolumeMounter()
		if err != nil {
			return nil, fmt.Errorf("failed to new volume mounter: %w", err)
		}
		mounter.RegisterMounter("volume", vm)
		// register tmpfs mounter
		mounter.RegisterMounter("tmpfs", docker.NewTmpfsMounter())
		rt, err := docker.NewRuntime(
			docker.WithMounter(mounter),
			docker.WithConfig(config.App.Executor.Docker.Config),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to new docker runtime: %w", err)
		}
		e.runtime = rt
	case runtime.Shell:
		e.runtime = shell.NewShellRuntime(shell.Config{
			CMD: config.App.Executor.Shell.CMD,
			UID: config.App.Executor.Shell.UID,
			GID: config.App.Executor.Shell.GID,
		})
	case runtime.Machine:
		rt, err := machine.NewRuntime(machine.WithConfig(machine.Config{
			Host:     config.App.Executor.Machine.Host,
			Port:     config.App.Executor.Machine.Port,
			Username: config.App.Executor.Machine.Username,
			Password: config.App.Executor.Machine.Password,
			HostKey:  config.App.Executor.Machine.HostKey,
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to new machine runtime: %w", err)
		}
		e.runtime = rt
	case runtime.Capability:
		e.runtime = capabilityruntime.New()
	default:
		return nil, fmt.Errorf("unknown runtime type: %s", e.runtimeType)
	}
	return e.runtime, nil
}

func (e *Engine) doRunTask(ctx context.Context, t *types.Task) error {
	// prepare limits
	if t.Limits == nil && (e.limits.DefaultCPUsLimit != "" || e.limits.DefaultMemoryLimit != "") {
		t.Limits = &types.TaskLimits{}
	}
	if t.Limits != nil && t.Limits.CPUs == "" {
		t.Limits.CPUs = e.limits.DefaultCPUsLimit
	}
	if t.Limits != nil && t.Limits.Memory == "" {
		t.Limits.Memory = e.limits.DefaultMemoryLimit
	}
	// create timeout context -- if timeout is defined
	rctx := ctx
	if t.Timeout != "" {
		dur, err := time.ParseDuration(t.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %s, %w", t.Timeout, err)
		}
		tctx, cancel := context.WithTimeout(ctx, dur)
		defer cancel()
		rctx = tctx
	}
	// run the task
	rtTask := t.Clone()
	if err := e.runtime.Run(rctx, rtTask); err != nil {
		finished := time.Now().UTC()
		t.FailedAt = &finished
		t.State = types.TaskStateFailed
		t.Error = err.Error()
		return fmt.Errorf("failed to run task: %w", err)
	}
	finished := time.Now().UTC()
	t.CompletedAt = &finished
	t.State = types.TaskStateCompleted
	t.Result = rtTask.Result
	return nil
}
