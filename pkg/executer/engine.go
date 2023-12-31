package executer

import (
	"context"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/executer/runtime"
	"github.com/flowline-io/flowbot/pkg/executer/runtime/docker"
	"github.com/flowline-io/flowbot/pkg/executer/runtime/shell"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Mode string

const (
	StateIdle      = "IDLE"
	StateRunning   = "RUNNING"
	StateCompleted = "COMPLETED"
)

type Engine struct {
	state    string
	mu       sync.Mutex
	mounters map[string]*runtime.MultiMounter
	runtime  runtime.Runtime
	limits   Limits
}

type Limits struct {
	DefaultCPUsLimit   string
	DefaultMemoryLimit string
}

func New() *Engine {
	return &Engine{
		state:    StateIdle,
		mounters: make(map[string]*runtime.MultiMounter),
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
		panic(errors.Errorf("engine is not %s", state))
	}
}

func (e *Engine) initRuntime() (runtime.Runtime, error) {
	e.limits = Limits{
		DefaultCPUsLimit:   config.App.Engine.Limits.Cpus,
		DefaultMemoryLimit: config.App.Engine.Limits.Memory,
	}
	runtimeType := runtime.Docker // default engine type
	switch runtimeType {
	case runtime.Docker:
		mounter, ok := e.mounters[runtime.Docker]
		if !ok {
			mounter = runtime.NewMultiMounter()
		}
		// register bind mounter
		bm := docker.NewBindMounter(docker.BindConfig{
			Allowed: config.App.Engine.Mounts.Bind.Allowed,
		})
		mounter.RegisterMounter("bind", bm)
		// register volume mounter
		vm, err := docker.NewVolumeMounter()
		if err != nil {
			return nil, err
		}
		mounter.RegisterMounter("volume", vm)
		// register tmpfs mounter
		mounter.RegisterMounter("tmpfs", docker.NewTmpfsMounter())
		rt, err := docker.NewDockerRuntime(
			docker.WithMounter(mounter),
			docker.WithConfig(config.App.Engine.Docker.Config),
		)
		if err != nil {
			return nil, err
		}
		e.runtime = rt
	case runtime.Shell:
		e.runtime = shell.NewShellRuntime(shell.Config{
			CMD: config.App.Engine.Shell.CMD,
			UID: config.App.Engine.Shell.UID,
			GID: config.App.Engine.Shell.GID,
		})
	default:
		return nil, errors.Errorf("unknown runtime type: %s", runtimeType)
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
			return errors.Wrapf(err, "invalid timeout duration: %s", t.Timeout)
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
		return nil
	}
	finished := time.Now().UTC()
	t.CompletedAt = &finished
	t.State = types.TaskStateCompleted
	t.Result = rtTask.Result
	e.state = StateCompleted
	return nil
}
