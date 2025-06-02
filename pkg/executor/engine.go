package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/docker"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/machine"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Mode string

const (
	StateIdle      = "IDLE"
	StateRunning   = "RUNNING"
	StateCompleted = "COMPLETED"
)

type Engine struct {
	state       string
	mu          sync.Mutex
	mounters    map[string]*runtime.MultiMounter
	runtime     runtime.Runtime
	limits      Limits
	runtimeType string
}

type Limits struct {
	DefaultCPUsLimit   string
	DefaultMemoryLimit string
}

func New(runtimeType string) *Engine {
	return &Engine{
		state:       StateIdle,
		mounters:    make(map[string]*runtime.MultiMounter),
		runtimeType: runtimeType,
	}
}

func (e *Engine) State() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

func (e *Engine) Run(ctx context.Context, t *types.Task) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mustState(StateIdle)
	e.state = StateRunning
	return e.runTask(ctx, t)
}

func (e *Engine) runTask(ctx context.Context, t *types.Task) error {
	if _, err := e.initRuntime(); err != nil {
		return err
	}

	return e.doRunTask(ctx, t)
}

func (e *Engine) mustState(state string) {
	if e.state != state {
		flog.Panic("engine is not %s", state)
	}
}

func (e *Engine) initRuntime() (runtime.Runtime, error) {
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
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to new machine runtime: %w", err)
		}
		e.runtime = rt
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
	e.state = StateCompleted
	return nil
}
